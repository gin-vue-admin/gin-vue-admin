// Package model 定义 GORM 实体与表结构。RBAC 核心：User/Role/Permission/Menu。
package model

import "time"

// Model 公共基础字段（不含软删除，按需在各实体追加 DeletedAt）。
type Model struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
