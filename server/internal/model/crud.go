package model

// CrudItem 通用增删改查示例实体（脚手架 CRUD 范例）。
// 字段沿用前端 crud demo 的"地址记录"语义，演示完整 CRUD + 分页 + 批量删除。
// 软删与审计字段（CreatedBy/UpdatedBy/DeletedBy）由嵌入的 Model 提供，GORM 回调自动注入。
// 新增业务模块时，复制本实体与对应 repository/service/handler 作为起点；
// 详见 docs/standards/ 「新增业务模块指南」。
type CrudItem struct {
	Model
	Date     string `gorm:"size:10" json:"date"`          // yyyy-MM-dd
	Name     string `gorm:"size:64;not null" json:"name"` // 列表搜索字段（模糊）
	Province string `gorm:"size:64" json:"province"`
	City     string `gorm:"size:64" json:"city"`
	Address  string `gorm:"size:255" json:"address"`
	Zip      int    `gorm:"default:0" json:"zip"`
}

// TableName 表名复数。
func (CrudItem) TableName() string { return "crud_items" }
