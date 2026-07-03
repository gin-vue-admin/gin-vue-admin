package repository

import (
	"context"

	"gorm.io/gorm"
	"gva/internal/model"
)

// MenuRepository 菜单数据访问。M4.1 只需 GetAllMenus。
type MenuRepository interface {
	GetAllMenus(ctx context.Context) ([]model.Menu, error)
}

type menuRepository struct {
	db *gorm.DB
}

func NewMenuRepository(db *gorm.DB) MenuRepository {
	return &menuRepository{db: db}
}

// GetAllMenus 查全部菜单，按 parent_id, sort 排序。
func (r *menuRepository) GetAllMenus(ctx context.Context) ([]model.Menu, error) {
	var menus []model.Menu
	err := r.db.WithContext(ctx).Order("parent_id, sort").Find(&menus).Error
	return menus, err
}
