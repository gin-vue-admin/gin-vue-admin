package repository

import (
	"context"

	"gorm.io/gorm"
	"gva/internal/model"
)

// MenuRepository 菜单数据访问。M4.1 只需 GetAllMenus。
type MenuRepository interface {
	GetAllMenus(ctx context.Context) ([]model.Menu, error)
	FindByID(ctx context.Context, id uint) (*model.Menu, error)
	Create(ctx context.Context, m *model.Menu) error
	Update(ctx context.Context, m *model.Menu) error
	DeleteByIDs(ctx context.Context, ids []uint) error          // 批量硬删（级联用）
	UpdateSorts(ctx context.Context, menus []model.Menu) error  // 批量更新 sort/parentId
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

// FindByID 按主键查菜单。
func (r *menuRepository) FindByID(ctx context.Context, id uint) (*model.Menu, error) {
	var m model.Menu
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// Create 新建菜单。
func (r *menuRepository) Create(ctx context.Context, m *model.Menu) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// Update 保存菜单（全字段覆盖）。
func (r *menuRepository) Update(ctx context.Context, m *model.Menu) error {
	return r.db.WithContext(ctx).Save(m).Error
}

// DeleteByIDs 批量硬删。
func (r *menuRepository) DeleteByIDs(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Delete(&model.Menu{}, ids).Error
}

// UpdateSorts 批量更新 sort/parentId（事务内逐条 Updates，菜单量小可接受）。
func (r *menuRepository) UpdateSorts(ctx context.Context, menus []model.Menu) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range menus {
			if err := tx.Model(&model.Menu{}).Where("id = ?", menus[i].ID).
				Updates(map[string]any{"sort": menus[i].Sort, "parent_id": menus[i].ParentID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
