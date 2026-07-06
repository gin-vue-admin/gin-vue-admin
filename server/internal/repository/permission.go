package repository

import (
	"context"

	"gva/internal/model"
	"gva/internal/pkg/pagination"

	"gorm.io/gorm"
)

// PermissionRepository 权限数据访问接口。
// 抽接口目的：PermissionService 及权限中间件需在单测中用 mock 替换 DB。
type PermissionRepository interface {
	List(ctx context.Context, q pagination.Query, module string) ([]model.Permission, int64, error)
	ListAll(ctx context.Context, q pagination.Query, module string) ([]model.Permission, error)
	FindByID(ctx context.Context, id uint) (*model.Permission, error)
	Create(ctx context.Context, p *model.Permission) error
	Update(ctx context.Context, p *model.Permission) error
	Delete(ctx context.Context, id uint) error
	BatchDelete(ctx context.Context, ids []uint) error
	GetUserPermissionCodes(ctx context.Context, userID uint) ([]string, error)
}

// permissionRepository gorm 实现。
type permissionRepository struct {
	db *gorm.DB
}

// NewPermissionRepository 构造权限仓储。
func NewPermissionRepository(db *gorm.DB) PermissionRepository {
	return &permissionRepository{db: db}
}

// applyFilters 叠加 module/status/keyword 模糊筛选。
// keyword 模糊匹配 name/code/description，module 与 status 精确匹配。
func applyFilters(db *gorm.DB, q pagination.Query, module string) *gorm.DB {
	if module != "" {
		db = db.Where("module = ?", module)
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		db = db.Where("name LIKE ? OR code LIKE ? OR description LIKE ?", like, like, like)
	}
	return db
}

// List 分页查询。count 与 list 各用独立 Session 隔离，避免 Where 叠加污染。
func (r *permissionRepository) List(ctx context.Context, q pagination.Query, module string) ([]model.Permission, int64, error) {
	var total int64
	countDB := applyFilters(r.db.WithContext(ctx).Session(&gorm.Session{}), q, module)
	if err := countDB.Model(&model.Permission{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var perms []model.Permission
	listDB := applyFilters(r.db.WithContext(ctx).Session(&gorm.Session{}), q, module)
	offset := (q.Page - 1) * q.Size
	if err := listDB.Offset(offset).Limit(q.Size).Find(&perms).Error; err != nil {
		return nil, 0, err
	}
	return perms, total, nil
}

// ListAll 不分页，返回全量（按 module/status/keyword 过滤后的全切片）。
func (r *permissionRepository) ListAll(ctx context.Context, q pagination.Query, module string) ([]model.Permission, error) {
	var perms []model.Permission
	db := applyFilters(r.db.WithContext(ctx), q, module)
	if err := db.Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

// FindByID 按主键查询，软删除记录自动过滤。
func (r *permissionRepository) FindByID(ctx context.Context, id uint) (*model.Permission, error) {
	var p model.Permission
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// Create 新建权限。
func (r *permissionRepository) Create(ctx context.Context, p *model.Permission) error {
	return r.db.WithContext(ctx).Create(p).Error
}

// Update 全量保存（Save 会更新所有字段）。
func (r *permissionRepository) Update(ctx context.Context, p *model.Permission) error {
	return r.db.WithContext(ctx).Save(p).Error
}

// Delete 软删（model.Permission 含 DeletedAt，GORM 自动写入 deleted_at）。
func (r *permissionRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Permission{}, id).Error
}

// BatchDelete 批量软删。
func (r *permissionRepository) BatchDelete(ctx context.Context, ids []uint) error {
	return r.db.WithContext(ctx).Delete(&model.Permission{}, ids).Error
}

// GetUserPermissionCodes 查用户所有角色的所有权限码。
// 跨表：user → user_roles → role_permissions → permissions。
// 标准 JOIN 语法，SQLite 与 MySQL 均兼容；排除软删的权限。
func (r *permissionRepository) GetUserPermissionCodes(ctx context.Context, userID uint) ([]string, error) {
	var codes []string
	err := r.db.WithContext(ctx).
		Raw(`SELECT DISTINCT p.code FROM permissions p
			JOIN role_permissions rp ON rp.permission_id = p.id
			JOIN user_roles ur ON ur.role_id = rp.role_id
			WHERE ur.user_id = ? AND p.deleted_at IS NULL`, userID).
		Scan(&codes).Error
	return codes, err
}
