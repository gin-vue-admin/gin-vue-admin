package repository

import (
	"context"

	"gva/internal/model"
	"gva/internal/pkg/pagination"

	"gorm.io/gorm"
)

// LoginLogQuery 登录日志查询参数。在 pagination.Query 基础上增加时间范围（复用 OperationLogQuery 风格）。
type LoginLogQuery struct {
	pagination.Query
	StartTime string // yyyy-MM-dd HH:mm:ss，闭区间
	EndTime   string
}

// LoginLogRepository 登录日志数据访问。
// 嵌入 GenericRepository 复用 FindByID/Delete/BatchDelete；List 支持 status/keyword(username/IP)/时间范围，Clear 硬删清空。
type LoginLogRepository interface {
	List(ctx context.Context, q LoginLogQuery) (pagination.Result[model.LoginLog], error)
	FindByID(ctx context.Context, id uint) (*model.LoginLog, error)
	Create(ctx context.Context, e *model.LoginLog) error
	Delete(ctx context.Context, id uint) error
	BatchDelete(ctx context.Context, ids []uint) error
	Clear(ctx context.Context) error // 清空全部（硬删）
}

type loginLogRepository struct {
	*GenericRepository[model.LoginLog]
}

func NewLoginLogRepository(db *gorm.DB) LoginLogRepository {
	return &loginLogRepository{GenericRepository: NewGenericRepository[model.LoginLog](db)}
}

func (r *loginLogRepository) List(ctx context.Context, q LoginLogQuery) (pagination.Result[model.LoginLog], error) {
	start, end := parseLogTimeRange(q.StartTime, q.EndTime)
	return r.GenericRepository.List(ctx, q.Query, func(db *gorm.DB) *gorm.DB {
		if q.Status != "" {
			db = db.Where("status = ?", q.Status)
		}
		if q.Keyword != "" {
			k := "%" + q.Keyword + "%"
			db = db.Where("username LIKE ? OR ip LIKE ?", k, k)
		}
		if !start.IsZero() {
			db = db.Where("created_at >= ?", start)
		}
		if !end.IsZero() {
			db = db.Where("created_at <= ?", end)
		}
		return db
	})
}

// Clear 硬删清空全部登录日志（日志无需保留软删痕迹）。
func (r *loginLogRepository) Clear(ctx context.Context) error {
	return r.DB(ctx).Unscoped().Where("1 = 1").Delete(&model.LoginLog{}).Error
}
