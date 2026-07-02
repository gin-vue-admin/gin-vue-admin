package model

import (
	"database/sql"

	"gorm.io/gorm"
)

// User 用户实体。
// 同时保留 Nickname（auth profile 用）与 RealName（用户列表用），兼容前端两处契约。
type User struct {
	Model
	Username    string         `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Password    string         `gorm:"size:128;not null" json:"-"` // 永不序列化
	Nickname    string         `gorm:"size:64" json:"nickname"`
	RealName    string         `gorm:"size:64" json:"realName"`
	Email       string         `gorm:"size:128" json:"email"`
	Phone       string         `gorm:"size:32" json:"phone"`
	Avatar      string         `gorm:"size:255" json:"avatar"`
	Status      string         `gorm:"size:16;default:active" json:"status"`
	LastLoginAt sql.NullTime   `gorm:"index" json:"-"`
	LoginCount  int            `gorm:"default:0" json:"loginCount"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Roles []Role `gorm:"many2many:user_roles;" json:"roles,omitempty"`
}

func (User) TableName() string { return "users" }

// Role 角色实体。super_admin 角色拥有通配权限。
type Role struct {
	Model
	Code        string         `gorm:"uniqueIndex;size:64;not null" json:"code"`
	Name        string         `gorm:"size:64;not null" json:"name"`
	Status      string         `gorm:"size:16;default:active" json:"status"`
	Sort        int            `gorm:"default:0" json:"sort"`
	Remark      string         `gorm:"size:255" json:"remark"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

func (Role) TableName() string { return "roles" }

// Permission 权限实体。code 如 user:read；super_admin 持有通配 *。
// Module/Description/Status 对齐前端 PermissionInfo 契约；DeletedAt 软删除。
type Permission struct {
	Model
	Code        string         `gorm:"uniqueIndex;size:128;not null" json:"code"`
	Name        string         `gorm:"size:64;not null" json:"name"`
	Type        string         `gorm:"size:16" json:"type"` // menu | button | api
	Module      string         `gorm:"size:32;index" json:"module"`
	Description string         `gorm:"size:255" json:"description"`
	Status      string         `gorm:"size:16;default:active" json:"status"` // active | inactive
	ParentID    uint           `gorm:"index;default:0" json:"parentId"`
	Sort        int            `gorm:"default:0" json:"sort"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Permission) TableName() string { return "permissions" }

// AutoMigrate 自动建表。M0/M1 用 AutoMigrate；M5 切换为 golang-migrate 版本化迁移。
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &Role{}, &Permission{}, &Menu{})
}
