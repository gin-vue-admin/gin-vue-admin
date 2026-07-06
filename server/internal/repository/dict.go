package repository

import (
	"context"

	"gva/internal/model"
	"gva/internal/pkg/pagination"

	"gorm.io/gorm"
)

// dict 三级（分类/字典/字典项）数据访问。每级嵌入 GenericRepository 复用 CRUD，
// 仅重写 List 注入过滤（keyword 多字段模糊 + status + 外键）。

// ===== Level 1: 字典分类 =====

// DictCategoryRepository 字典分类仓储。
type DictCategoryRepository interface {
	List(ctx context.Context, q pagination.Query) (pagination.Result[model.DictCategory], error)
	FindByID(ctx context.Context, id uint) (*model.DictCategory, error)
	Create(ctx context.Context, e *model.DictCategory) error
	Update(ctx context.Context, e *model.DictCategory) error
	Delete(ctx context.Context, id uint) error
	BatchDelete(ctx context.Context, ids []uint) error
}

type dictCategoryRepository struct {
	*GenericRepository[model.DictCategory]
}

func NewDictCategoryRepository(db *gorm.DB) DictCategoryRepository {
	return &dictCategoryRepository{GenericRepository: NewGenericRepository[model.DictCategory](db)}
}

func (r *dictCategoryRepository) List(ctx context.Context, q pagination.Query) (pagination.Result[model.DictCategory], error) {
	return r.GenericRepository.List(ctx, q, func(db *gorm.DB) *gorm.DB {
		if q.Status != "" {
			db = db.Where("status = ?", q.Status)
		}
		if q.Keyword != "" {
			k := "%" + q.Keyword + "%"
			db = db.Where("name LIKE ? OR code LIKE ? OR description LIKE ?", k, k, k)
		}
		return db
	})
}

// ===== Level 2: 字典 =====

// DictRepository 字典仓储。List 额外按 categoryId 过滤。
type DictRepository interface {
	List(ctx context.Context, q pagination.Query, categoryID uint) (pagination.Result[model.Dict], error)
	FindByID(ctx context.Context, id uint) (*model.Dict, error)
	Create(ctx context.Context, e *model.Dict) error
	Update(ctx context.Context, e *model.Dict) error
	Delete(ctx context.Context, id uint) error
	BatchDelete(ctx context.Context, ids []uint) error
}

type dictRepository struct {
	*GenericRepository[model.Dict]
}

func NewDictRepository(db *gorm.DB) DictRepository {
	return &dictRepository{GenericRepository: NewGenericRepository[model.Dict](db)}
}

func (r *dictRepository) List(ctx context.Context, q pagination.Query, categoryID uint) (pagination.Result[model.Dict], error) {
	return r.GenericRepository.List(ctx, q, func(db *gorm.DB) *gorm.DB {
		if categoryID > 0 {
			db = db.Where("category_id = ?", categoryID)
		}
		if q.Status != "" {
			db = db.Where("status = ?", q.Status)
		}
		if q.Keyword != "" {
			k := "%" + q.Keyword + "%"
			db = db.Where("name LIKE ? OR code LIKE ? OR description LIKE ?", k, k, k)
		}
		return db
	})
}

// ===== Level 3: 字典项 =====

// DictItemRepository 字典项仓储。List 额外按 dictId 过滤。
type DictItemRepository interface {
	List(ctx context.Context, q pagination.Query, dictID uint) (pagination.Result[model.DictItem], error)
	FindByID(ctx context.Context, id uint) (*model.DictItem, error)
	Create(ctx context.Context, e *model.DictItem) error
	Update(ctx context.Context, e *model.DictItem) error
	Delete(ctx context.Context, id uint) error
	BatchDelete(ctx context.Context, ids []uint) error
}

type dictItemRepository struct {
	*GenericRepository[model.DictItem]
}

func NewDictItemRepository(db *gorm.DB) DictItemRepository {
	return &dictItemRepository{GenericRepository: NewGenericRepository[model.DictItem](db)}
}

func (r *dictItemRepository) List(ctx context.Context, q pagination.Query, dictID uint) (pagination.Result[model.DictItem], error) {
	return r.GenericRepository.List(ctx, q, func(db *gorm.DB) *gorm.DB {
		if dictID > 0 {
			db = db.Where("dict_id = ?", dictID)
		}
		if q.Status != "" {
			db = db.Where("status = ?", q.Status)
		}
		if q.Keyword != "" {
			k := "%" + q.Keyword + "%"
			db = db.Where("name LIKE ? OR code LIKE ? OR value LIKE ?", k, k, k)
		}
		return db
	})
}
