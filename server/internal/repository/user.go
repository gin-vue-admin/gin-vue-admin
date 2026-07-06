// Package repository 封装数据访问。仅对含分支业务逻辑的领域抽接口（为单测可注入 mock）。
package repository

import (
	"context"
	"time"

	"gva/internal/model"
	"gva/internal/pkg/datascope"
	"gva/internal/pkg/pagination"

	"gorm.io/gorm"
)

// UserRepository 用户数据访问接口。
// 抽接口目的：auth service 含登录/刷新等分支逻辑，需在单测中用 mock 替换 DB。
// M3.3 扩展：在 M2 4 方法（登录/资料场景）基础上补列表/CRUD/角色分配，供用户管理域使用。
type UserRepository interface {
	// M2 既有
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	FindByID(ctx context.Context, id uint) (*model.User, error)
	FindByIDWithRoles(ctx context.Context, id uint) (*model.User, error)
	UpdateLoginStats(ctx context.Context, id uint) error
	// M3.3 新增
	List(ctx context.Context, q pagination.Query, roleCode string, scope datascope.Scope) ([]model.User, int64, error)
	ListAll(ctx context.Context, q pagination.Query, roleCode string, scope datascope.Scope) ([]model.User, error) // 不分页，导出用
	Create(ctx context.Context, u *model.User) error
	Update(ctx context.Context, u *model.User) error
	Delete(ctx context.Context, id uint) error
	BatchDelete(ctx context.Context, ids []uint) error
	ReplaceRoles(ctx context.Context, userID uint, roleIDs []uint) error
	FindRoleIDsByCodes(ctx context.Context, codes []string) (map[string]uint, error)
}

// userRepository gorm 实现。
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 构造用户仓储。
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var u model.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) FindByID(ctx context.Context, id uint) (*model.User, error) {
	var u model.User
	if err := r.db.WithContext(ctx).First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// FindByIDWithRoles 预加载 Roles 及其 Permissions（双层），供 GetProfile 构造 profile。
func (r *userRepository) FindByIDWithRoles(ctx context.Context, id uint) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).
		Preload("Roles.Permissions").
		First(&u, id).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdateLoginStats 更新最后登录时间与登录次数。失败仅由调用方告警，不影响登录主流程。
func (r *userRepository) UpdateLoginStats(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).
		Updates(map[string]any{
			"last_login_at": time.Now(),
			"login_count":   gorm.Expr("login_count + 1"),
		}).Error
}

// applyUserFilters 叠加 status/keyword/roleCode/数据范围 过滤。
// 多表 JOIN（按角色过滤）时 Where 字段必须带表名前缀（users.status/users.username...），
// 否则与 roles/user_roles 表的同名字段会引发 ambiguous column 错误。
// 命名为 applyUserFilters，避免与 role.go 的 applyRoleFilters、permission.go 的 applyFilters 冲突。
// scope 对 User 实体：部门列 users.dept_id，"本人"列 users.id（即只能看到自己这条）。
func applyUserFilters(db *gorm.DB, q pagination.Query, roleCode string, scope datascope.Scope) *gorm.DB {
	if q.Status != "" {
		db = db.Where("users.status = ?", q.Status)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		db = db.Where("users.username LIKE ? OR users.real_name LIKE ? OR users.email LIKE ? OR users.phone LIKE ?", like, like, like, like)
	}
	if roleCode != "" {
		db = db.Joins("JOIN user_roles ur ON ur.user_id = users.id").
			Joins("JOIN roles r ON r.id = ur.role_id").
			Where("r.code = ?", roleCode)
	}
	db = scope.Apply(db, "users.dept_id", "users.id")
	return db
}

// List 分页查询用户。count 与 list 各用独立 Session 隔离，避免 Where/Joins 叠加污染。
// roleCode 非空时通过 user_roles/roles JOIN 过滤；scope 叠加数据权限；Preload Roles 供前端列表展示。
func (r *userRepository) List(ctx context.Context, q pagination.Query, roleCode string, scope datascope.Scope) ([]model.User, int64, error) {
	var total int64
	countDB := applyUserFilters(r.db.WithContext(ctx).Session(&gorm.Session{}), q, roleCode, scope)
	if err := countDB.Model(&model.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []model.User
	listDB := applyUserFilters(r.db.WithContext(ctx).Session(&gorm.Session{}), q, roleCode, scope).Preload("Roles")
	offset := (q.Page - 1) * q.Size
	if err := listDB.Offset(offset).Limit(q.Size).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// ListAll 不分页返全量（按 status/keyword/roleCode/scope 过滤后的全切片），导出用。
func (r *userRepository) ListAll(ctx context.Context, q pagination.Query, roleCode string, scope datascope.Scope) ([]model.User, error) {
	var users []model.User
	db := applyUserFilters(r.db.WithContext(ctx), q, roleCode, scope).Preload("Roles")
	if err := db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// Create 新建用户。
func (r *userRepository) Create(ctx context.Context, u *model.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

// Update 全量保存（Save 会更新所有字段）。
func (r *userRepository) Update(ctx context.Context, u *model.User) error {
	return r.db.WithContext(ctx).Save(u).Error
}

// Delete 事务内软删用户 + 清 user_roles 关联。
// User 未声明反向给 Role 的关联（Role 上无 Users 字段），故 user_roles 必须用 raw SQL 清，
// 不能用 Association。raw SQL 兼容 SQLite 与 MySQL。
func (r *userRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.User{}, id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM user_roles WHERE user_id = ?", id).Error; err != nil {
			return err
		}
		return nil
	})
}

// BatchDelete 批量软删 + 清 user_roles 关联。空 ids 直接返回，避免 IN () 语法错误。
func (r *userRepository) BatchDelete(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.User{}, ids).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM user_roles WHERE user_id IN ?", ids).Error; err != nil {
			return err
		}
		return nil
	})
}

// ReplaceRoles 全量替换用户角色（Association Replace）。
// roleIDs 为空时 roles 为空切片，Replace 清空所有角色。
func (r *userRepository) ReplaceRoles(ctx context.Context, userID uint, roleIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var u model.User
		if err := tx.First(&u, userID).Error; err != nil {
			return err
		}
		var roles []model.Role
		if len(roleIDs) > 0 {
			if err := tx.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
				return err
			}
		}
		return tx.Model(&u).Association("Roles").Replace(&roles)
	})
}

// FindRoleIDsByCodes 按 code 批量查角色 id（未知 code 不在 map）。
// 用于创建/更新用户时把前端传入的 role code 数组转为 id 数组。
func (r *userRepository) FindRoleIDsByCodes(ctx context.Context, codes []string) (map[string]uint, error) {
	m := make(map[string]uint)
	if len(codes) == 0 {
		return m, nil
	}
	var roles []model.Role
	if err := r.db.WithContext(ctx).Where("code IN ?", codes).Find(&roles).Error; err != nil {
		return nil, err
	}
	for _, ro := range roles {
		m[ro.Code] = ro.ID
	}
	return m, nil
}
