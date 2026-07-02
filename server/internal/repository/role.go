package repository

import (
	"context"

	"gorm.io/gorm"
	"gva/internal/model"
	"gva/internal/pkg/pagination"
)

// RoleRepository 角色数据访问接口。
// 含权限分配（Replace/Get/FindByCodes）与删除时解除关联（role_permissions/user_roles）。
type RoleRepository interface {
	List(ctx context.Context, q pagination.Query) ([]model.Role, int64, error)
	ListAll(ctx context.Context, q pagination.Query) ([]model.Role, error) // 不分页，导出用
	FindByID(ctx context.Context, id uint) (*model.Role, error)
	Create(ctx context.Context, r *model.Role) error
	Update(ctx context.Context, r *model.Role) error
	Delete(ctx context.Context, id uint) error // 事务内软删+解除关联
	BatchDelete(ctx context.Context, ids []uint) error
	GetRolePermissionCodes(ctx context.Context, roleID uint) ([]string, error)
	ReplaceRolePermissions(ctx context.Context, roleID uint, permissionIDs []uint) error
	FindPermissionIDsByCodes(ctx context.Context, codes []string) (map[string]uint, error)
}

// roleRepository gorm 实现。
type roleRepository struct {
	db *gorm.DB
}

// NewRoleRepository 构造角色仓储。
func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{db: db}
}

// applyRoleFilters 叠加 status/keyword 模糊筛选（角色无 module 概念）。
// keyword 模糊匹配 name/code/description，status 精确匹配。
// 命名为 applyRoleFilters，避免与 permission.go 的 applyFilters 冲突。
func applyRoleFilters(db *gorm.DB, q pagination.Query) *gorm.DB {
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
func (r *roleRepository) List(ctx context.Context, q pagination.Query) ([]model.Role, int64, error) {
	var total int64
	countDB := applyRoleFilters(r.db.WithContext(ctx).Session(&gorm.Session{}), q)
	if err := countDB.Model(&model.Role{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var roles []model.Role
	listDB := applyRoleFilters(r.db.WithContext(ctx).Session(&gorm.Session{}), q)
	offset := (q.Page - 1) * q.Size
	if err := listDB.Offset(offset).Limit(q.Size).Find(&roles).Error; err != nil {
		return nil, 0, err
	}
	return roles, total, nil
}

// ListAll 不分页返全量（按 status/keyword 过滤后的全切片），导出用。
func (r *roleRepository) ListAll(ctx context.Context, q pagination.Query) ([]model.Role, error) {
	var roles []model.Role
	db := applyRoleFilters(r.db.WithContext(ctx), q)
	if err := db.Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// FindByID 按主键查询，软删除记录自动过滤。
func (r *roleRepository) FindByID(ctx context.Context, id uint) (*model.Role, error) {
	var role model.Role
	if err := r.db.WithContext(ctx).First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// Create 新建角色。
func (r *roleRepository) Create(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

// Update 全量保存（Save 会更新所有字段）。
func (r *roleRepository) Update(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

// Delete 事务内软删角色 + 解除 role_permissions 与 user_roles 关联。
// model.Role 声明了 Permissions 反向关联，可用 Association；但 User 未声明反向给 Role 的
// 关联（Role 上无 Users 字段），故 user_roles 必须用 raw SQL 清。
// 为统一可靠，两张关联表均用 tx.Exec raw SQL 删除（GORM 对 &struct{}{} 支持不稳定）。
func (r *roleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.Role{}, id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM role_permissions WHERE role_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM user_roles WHERE role_id = ?", id).Error; err != nil {
			return err
		}
		return nil
	})
}

// BatchDelete 批量软删 + 解除关联。raw SQL 用 role_id IN ? 清两张关联表。
func (r *roleRepository) BatchDelete(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.Role{}, ids).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM role_permissions WHERE role_id IN ?", ids).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM user_roles WHERE role_id IN ?", ids).Error; err != nil {
			return err
		}
		return nil
	})
}

// GetRolePermissionCodes 查角色已分配的权限 code 数组。
// 跨表 role → role_permissions → permissions；排除软删的权限。
// 同 M3.1 GetUserPermissionCodes 模式，标准 JOIN 兼容 SQLite 与 MySQL。
func (r *roleRepository) GetRolePermissionCodes(ctx context.Context, roleID uint) ([]string, error) {
	var codes []string
	err := r.db.WithContext(ctx).
		Raw(`SELECT DISTINCT p.code FROM permissions p
			JOIN role_permissions rp ON rp.permission_id = p.id
			WHERE rp.role_id = ? AND p.deleted_at IS NULL`, roleID).
		Scan(&codes).Error
	return codes, err
}

// ReplaceRolePermissions 全量替换角色权限。
// permissionIDs 为空时 perms 为空切片，Association.Replace 清空所有权限。
func (r *roleRepository) ReplaceRolePermissions(ctx context.Context, roleID uint, permissionIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role model.Role
		if err := tx.First(&role, roleID).Error; err != nil {
			return err
		}
		var perms []model.Permission
		if len(permissionIDs) > 0 {
			if err := tx.Where("id IN ?", permissionIDs).Find(&perms).Error; err != nil {
				return err
			}
		}
		return tx.Model(&role).Association("Permissions").Replace(&perms)
	})
}

// FindPermissionIDsByCodes 按 code 批量查 id，返回存在的 code→id 映射（未知 code 不在 map）。
func (r *roleRepository) FindPermissionIDsByCodes(ctx context.Context, codes []string) (map[string]uint, error) {
	m := make(map[string]uint)
	if len(codes) == 0 {
		return m, nil
	}
	var perms []model.Permission
	if err := r.db.WithContext(ctx).Where("code IN ?", codes).Find(&perms).Error; err != nil {
		return nil, err
	}
	for _, p := range perms {
		m[p.Code] = p.ID
	}
	return m, nil
}
