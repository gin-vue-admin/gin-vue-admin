// Package repository notice 数据访问（复用 GenericRepository）。
package repository

import (
	"context"

	"gva/internal/model"
	"gva/internal/pkg/pagination"

	"gorm.io/gorm"
)

// NoticeRepository 公告数据访问接口。
type NoticeRepository interface {
	List(ctx context.Context, q pagination.Query, noticeType string) (pagination.Result[model.Notice], error)
	FindByID(ctx context.Context, id uint) (*model.Notice, error)
	Create(ctx context.Context, n *model.Notice) error
	Update(ctx context.Context, n *model.Notice) error
	Delete(ctx context.Context, id uint) error
	BatchDelete(ctx context.Context, ids []uint) error
}

type noticeRepository struct {
	*GenericRepository[model.Notice]
}

// NewNoticeRepository 构造公告仓库。
func NewNoticeRepository(db *gorm.DB) NoticeRepository {
	return &noticeRepository{GenericRepository: NewGenericRepository[model.Notice](db)}
}

func (r *noticeRepository) List(ctx context.Context, q pagination.Query, noticeType string) (pagination.Result[model.Notice], error) {
	return r.GenericRepository.List(ctx, q, func(db *gorm.DB) *gorm.DB {
		if q.Keyword != "" {
			db = db.Where("title LIKE ?", "%"+q.Keyword+"%")
		}
		if q.Status != "" {
			db = db.Where("status = ?", q.Status)
		}
		if noticeType != "" {
			db = db.Where("type = ?", noticeType)
		}
		return db
	})
}
