package repository

import (
	"context"

	"gva/internal/model"
	"gva/internal/pkg/pagination"

	"gorm.io/gorm"
)

// SysConfigRepository 系统参数配置仓储接口。
type SysConfigRepository interface {
	List(ctx context.Context, q pagination.Query) (pagination.Result[model.SysConfig], error)
	FindByID(ctx context.Context, id uint) (*model.SysConfig, error)
	FindByKey(ctx context.Context, key string) (*model.SysConfig, error)
	FindAll(ctx context.Context) ([]model.SysConfig, error)
	Create(ctx context.Context, e *model.SysConfig) error
	Update(ctx context.Context, e *model.SysConfig) error
	Delete(ctx context.Context, id uint) error
}

// sysConfigRepository 嵌入 GenericRepository 复用 CRUD，仅重写 List 注入 keyword 过滤 + FindByKey/FindAll。
type sysConfigRepository struct {
	*GenericRepository[model.SysConfig]
}

func NewSysConfigRepository(db *gorm.DB) SysConfigRepository {
	return &sysConfigRepository{GenericRepository: NewGenericRepository[model.SysConfig](db)}
}

// List 分页查询，叠加 config_key / config_name 模糊过滤。
func (r *sysConfigRepository) List(ctx context.Context, q pagination.Query) (pagination.Result[model.SysConfig], error) {
	return r.GenericRepository.List(ctx, q, func(db *gorm.DB) *gorm.DB {
		if q.Keyword != "" {
			return db.Where("config_key LIKE ? OR config_name LIKE ?", "%"+q.Keyword+"%", "%"+q.Keyword+"%")
		}
		return db
	})
}

// FindByKey 按 config_key 查询（唯一）。未找到返回 gorm.ErrRecordNotFound。
func (r *sysConfigRepository) FindByKey(ctx context.Context, key string) (*model.SysConfig, error) {
	var c model.SysConfig
	if err := r.DB(ctx).Where("config_key = ?", key).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// FindAll 取全量（启动加载缓存用）。
func (r *sysConfigRepository) FindAll(ctx context.Context) ([]model.SysConfig, error) {
	return r.ListAll(ctx, nil)
}
