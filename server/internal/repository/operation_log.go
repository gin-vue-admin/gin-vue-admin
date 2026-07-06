package repository

import (
	"context"
	"time"

	"gva/internal/model"
	"gva/internal/pkg/pagination"

	"gorm.io/gorm"
)

// OperationLogQuery 操作日志查询参数。在 pagination.Query 基础上增加时间范围。
type OperationLogQuery struct {
	pagination.Query
	StartTime string // yyyy-MM-dd HH:mm:ss，闭区间
	EndTime   string
}

// OperationLogRepository 操作日志数据访问。
// 嵌入 GenericRepository 复用 FindByID/Delete/BatchDelete；List 支持时间范围，Clear 硬删清空。
type OperationLogRepository interface {
	List(ctx context.Context, q OperationLogQuery) (pagination.Result[model.OperationLog], error)
	FindByID(ctx context.Context, id uint) (*model.OperationLog, error)
	Create(ctx context.Context, e *model.OperationLog) error
	Delete(ctx context.Context, id uint) error
	BatchDelete(ctx context.Context, ids []uint) error
	Clear(ctx context.Context) error // 清空全部（硬删）
}

type operationLogRepository struct {
	*GenericRepository[model.OperationLog]
}

func NewOperationLogRepository(db *gorm.DB) OperationLogRepository {
	return &operationLogRepository{GenericRepository: NewGenericRepository[model.OperationLog](db)}
}

func (r *operationLogRepository) List(ctx context.Context, q OperationLogQuery) (pagination.Result[model.OperationLog], error) {
	start, end := parseLogTimeRange(q.StartTime, q.EndTime)
	return r.GenericRepository.List(ctx, q.Query, func(db *gorm.DB) *gorm.DB {
		if q.Status != "" {
			db = db.Where("status = ?", q.Status)
		}
		if q.Keyword != "" {
			k := "%" + q.Keyword + "%"
			db = db.Where("username LIKE ? OR path LIKE ?", k, k)
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

// Clear 硬删清空全部操作日志（日志无需保留软删痕迹）。
func (r *operationLogRepository) Clear(ctx context.Context) error {
	return r.DB(ctx).Unscoped().Where("1 = 1").Delete(&model.OperationLog{}).Error
}

// parseLogTimeRange 解析 yyyy-MM-dd HH:mm:ss 时间范围；空串返回零值。
func parseLogTimeRange(start, end string) (time.Time, time.Time) {
	const layout = "2006-01-02 15:04:05"
	var s, e time.Time
	if start != "" {
		s, _ = time.Parse(layout, start)
	}
	if end != "" {
		e, _ = time.Parse(layout, end)
	}
	return s, e
}
