package model

// Dept 部门实体（树形自关联，ParentID 指向父部门，0=根）。
// 是数据权限（data scope）的组织基础；用户归属部门决定其可见数据范围（M7 后续）。
// 软删与审计字段（CreatedBy/UpdatedBy/DeletedBy）由嵌入的 Model 提供。
type Dept struct {
	Model
	ParentID uint   `gorm:"index;default:0" json:"parentId"`
	Name     string `gorm:"size:64;not null" json:"name"`
	Leader   string `gorm:"size:64" json:"leader"`
	Phone    string `gorm:"size:32" json:"phone"`
	Email    string `gorm:"size:128" json:"email"`
	Sort     int    `gorm:"default:0" json:"sort"`
	Status   string `gorm:"size:16;default:active" json:"status"`
}

// TableName 表名复数。
func (Dept) TableName() string { return "depts" }
