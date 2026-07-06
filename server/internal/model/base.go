// Package model 定义 GORM 实体与表结构。RBAC 核心：User/Role/Permission/Menu + CrudItem 范例。
package model

import (
	"time"

	"gorm.io/gorm"
)

// Model 公共基础字段：主键 + 时间戳 + 审计字段 + 软删除。
//
// 审计字段 CreatedBy/UpdatedBy/DeletedBy 记录操作者 user.id，由 internal/pkg/audit 的
// GORM 回调从请求上下文（middleware.AuthRequired 注入）自动填充，实现操作者追溯。
// - CreatedBy/UpdatedBy：Create/Update 回调自动写（M6）
// - DeletedBy：软删时由 M8 操作日志统一写入（字段先占位）
//
// JSON tag 对齐前端契约：id/createTime/updateTime/createdBy/updatedBy；
// 软删字段（deletedAt/deletedBy）不序列化。
type Model struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createTime"`
	UpdatedAt time.Time      `json:"updateTime"`
	CreatedBy uint           `gorm:"index" json:"createdBy"`
	UpdatedBy uint           `gorm:"index" json:"updatedBy"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	DeletedBy uint           `gorm:"index;default:0" json:"-"`
}
