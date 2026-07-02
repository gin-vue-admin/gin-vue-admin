// Package pagination 通用分页：Query/Result + 泛型 Paginate。三域列表复用。
package pagination

import "gorm.io/gorm"

// Query 通用列表查询参数（对齐前端 PermissionSearchRequest 公共字段）。
type Query struct {
	Keyword string `form:"keyword"`
	Status  string `form:"status"`
	Page    int    `form:"page,default=1"`
	Size    int    `form:"size,default=10"`
}

const maxPageSize = 100

// Normalize 补默认值并限制 size 上限。
func (q *Query) Normalize() {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Size < 1 {
		q.Size = 10
	}
	if q.Size > maxPageSize {
		q.Size = maxPageSize
	}
}

// Result 分页响应（对齐前端 {records,total,current,size}）。
type Result[T any] struct {
	Records []T   `json:"records"`
	Total   int64 `json:"total"`
	Current int   `json:"current"`
	Size    int   `json:"size"`
}

// Paginate 在 build 叠加的查询基础上做 count + 分页，返回 Result。
// build 回调用于叠加域特有 Where（如 module、keyword 模糊匹配）。
func Paginate[T any](db *gorm.DB, q Query, build func(*gorm.DB) *gorm.DB) (Result[T], error) {
	var total int64
	countDB := build(db.Session(&gorm.Session{}))
	if err := countDB.Model(new(T)).Count(&total).Error; err != nil {
		return Result[T]{}, err
	}
	var records []T
	listDB := build(db.Session(&gorm.Session{}))
	offset := (q.Page - 1) * q.Size
	if err := listDB.Offset(offset).Limit(q.Size).Find(&records).Error; err != nil {
		return Result[T]{}, err
	}
	return Result[T]{
		Records: records,
		Total:   total,
		Current: q.Page,
		Size:    q.Size,
	}, nil
}
