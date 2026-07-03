package model

// Menu 菜单实体，对应前端动态路由 / 菜单树。
// 通过 PermissionCode 与 RBAC 权限码关联，决定可见性。
type Menu struct {
	Model
	ParentID       uint   `gorm:"index;default:0" json:"parentId"`
	Name           string `gorm:"size:64" json:"name"`          // 路由名（如 systemUser）
	Title          string `gorm:"size:64" json:"title"`         // 显示名（如 用户管理）
	Path           string `gorm:"size:255" json:"path"`
	Component      string `gorm:"size:255" json:"component"`
	Icon           string `gorm:"size:64" json:"icon"` // 前端要求 PascalCase 全局唯一
	Sort           int    `gorm:"default:0" json:"sort"`
	ShowMenu       bool   `gorm:"default:true" json:"showMenu"`
	PermissionCode string `gorm:"size:128" json:"permissionCode"`
}

func (Menu) TableName() string { return "menus" }
