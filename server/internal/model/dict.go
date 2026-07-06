package model

// DictCategory 字典分类（三级层级 level 1）。
// 软删与审计由嵌入的 Model 提供。code 用于程序化引用。
type DictCategory struct {
	Model
	Name        string `gorm:"size:64;not null" json:"name"`
	Code        string `gorm:"size:64;not null" json:"code"`
	Description string `gorm:"size:255" json:"description"`
	Status      string `gorm:"size:16;default:active" json:"status"`
}

func (DictCategory) TableName() string { return "dict_categories" }

// Dict 字典（level 2，归属分类）。CategoryID 指向 DictCategory.ID。
type Dict struct {
	Model
	CategoryID  uint   `gorm:"index;not null" json:"categoryId"`
	Name        string `gorm:"size:64;not null" json:"name"`
	Code        string `gorm:"size:64;not null" json:"code"`
	Description string `gorm:"size:255" json:"description"`
	Status      string `gorm:"size:16;default:active" json:"status"`
}

func (Dict) TableName() string { return "dicts" }

// DictItem 字典项（level 3，归属字典）。DictID 指向 Dict.ID。
type DictItem struct {
	Model
	DictID uint   `gorm:"index;not null" json:"dictId"`
	Name   string `gorm:"size:64;not null" json:"name"`
	Code   string `gorm:"size:64;not null" json:"code"`
	Value  string `gorm:"size:255" json:"value"`
	Sort   int    `gorm:"default:0" json:"sort"`
	Status string `gorm:"size:16;default:active" json:"status"`
}

func (DictItem) TableName() string { return "dict_items" }
