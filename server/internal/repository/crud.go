package repository

import (
	"context"

	"gva/internal/model"
	"gva/internal/pkg/pagination"

	"gorm.io/gorm"
)

// CrudRepository 通用 CRUD 示例仓储接口（对齐前端 /api/crud 契约：列表+详情+增改+单删+批删）。
// 演示新模块如何通过嵌入 GenericRepository 复用样板，仅声明模块特有的过滤逻辑。
type CrudRepository interface {
	List(ctx context.Context, q pagination.Query) (pagination.Result[model.CrudItem], error)
	FindByID(ctx context.Context, id uint) (*model.CrudItem, error)
	Create(ctx context.Context, e *model.CrudItem) error
	Update(ctx context.Context, e *model.CrudItem) error
	Delete(ctx context.Context, id uint) error
	BatchDelete(ctx context.Context, ids []uint) error
}

// crudRepository 嵌入 GenericRepository[model.CrudItem]：
// FindByID/Create/Update/Delete/BatchDelete 全部由基类提供，本类型只重写 List 注入 keyword 过滤。
type crudRepository struct {
	*GenericRepository[model.CrudItem]
}

// NewCrudRepository 构造 crud 仓储。
func NewCrudRepository(db *gorm.DB) CrudRepository {
	return &crudRepository{GenericRepository: NewGenericRepository[model.CrudItem](db)}
}

// List 分页查询，叠加 name keyword 模糊过滤。
func (r *crudRepository) List(ctx context.Context, q pagination.Query) (pagination.Result[model.CrudItem], error) {
	return r.GenericRepository.List(ctx, q, func(db *gorm.DB) *gorm.DB {
		if q.Keyword != "" {
			return db.Where("name LIKE ?", "%"+q.Keyword+"%")
		}
		return db
	})
}
