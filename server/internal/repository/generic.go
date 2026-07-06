// Package repository 数据访问层。
//
// GenericRepository 为通用 CRUD + 分页基类，消除各业务模块 repository 的同构样板。
// T 为 GORM 实体【值类型】（如 CrudItem），内部用 new(T) 取指针做 GORM 操作，
// 并复用 pagination.Paginate[T] 做分页（对齐前端 {records,total,current,size} 契约）。
//
// 用法：模块 repository 嵌入本基类，只写差异——applyXxxFilters 过滤函数与关联清理 hook。
// 详见 docs/standards/ 「新增业务模块指南」。
package repository

import (
	"context"
	"reflect"
	"sync"
	"time"

	"gva/internal/pkg/audit"
	"gva/internal/pkg/pagination"

	"gorm.io/gorm"
)

// softDeleteTypeCache 缓存「T 是否含 gorm.DeletedAt 字段」按 reflect.Type 缓存，避免每次删除都解析。
var softDeleteTypeCache sync.Map

// isSoftDelete 反射检测 T 是否含 gorm.DeletedAt 字段（含嵌入 model.Model 的提升字段）。
// 决定 Delete/BatchDelete 走软删 Updates（写 deleted_at+deleted_by）还是硬删。
func isSoftDelete[T any]() bool {
	var zero T
	rt := reflect.TypeOf(&zero).Elem()
	if v, ok := softDeleteTypeCache.Load(rt); ok {
		return v.(bool)
	}
	dt := reflect.TypeOf(gorm.DeletedAt{})
	has := false
	var walk func(reflect.Type)
	walk = func(t reflect.Type) {
		if has || t.Kind() != reflect.Struct {
			return
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.Type == dt {
				has = true
				return
			}
			if f.Anonymous {
				walk(f.Type)
			}
		}
	}
	walk(rt)
	softDeleteTypeCache.Store(rt, has)
	return has
}

// softDeleteUpdates 构造软删 UPDATE 的 map：始终写 deleted_at，ctx 含 userID 时追加 deleted_by。
func softDeleteUpdates(ctx context.Context) map[string]any {
	updates := map[string]any{"deleted_at": time.Now()}
	if uid, ok := audit.UserIDFrom(ctx); ok {
		updates["deleted_by"] = uid
	}
	return updates
}

// FilterFunc 让调用方叠加模块特有的 Where/Preload（keyword 模糊、状态过滤、关联预加载等）。
// 签名与 pagination.Paginate 的 build 回调一致，可直接透传。
// List 的 count 也会经过此函数，Preload 不影响 Count，可安全使用。
type FilterFunc func(*gorm.DB) *gorm.DB

// GenericRepository 通用 CRUD + 分页基类。T 为 GORM 实体值类型。
type GenericRepository[T any] struct {
	db *gorm.DB
}

// NewGenericRepository 构造泛型仓储。仅传入 db；实体类型由 T 推导，主键/表名通过 new(T) 推断。
func NewGenericRepository[T any](db *gorm.DB) *GenericRepository[T] {
	return &GenericRepository[T]{db: db}
}

// DB 暴露带 ctx 的 *gorm.DB，供模块编写基类未覆盖的特有查询（如关联替换、聚合）。
func (r *GenericRepository[T]) DB(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

// FindByID 按主键查询，软删记录自动过滤。未找到返回 gorm.ErrRecordNotFound。
func (r *GenericRepository[T]) FindByID(ctx context.Context, id uint) (*T, error) {
	e := new(T)
	if err := r.db.WithContext(ctx).First(e, id).Error; err != nil {
		return nil, err
	}
	return e, nil
}

// List 分页查询，复用 pagination.Paginate。filter 叠加模块特有 Where/Preload（可为 nil）。
// 返回 Result[T]，对齐前端 {records,total,current,size} 契约。
func (r *GenericRepository[T]) List(ctx context.Context, q pagination.Query, filter FilterFunc) (pagination.Result[T], error) {
	build := filter
	if build == nil {
		build = func(db *gorm.DB) *gorm.DB { return db }
	}
	return pagination.Paginate[T](r.db.WithContext(ctx), q, build)
}

// ListAll 不分页返全量（过滤后），导出 CSV 用。filter 可为 nil。
func (r *GenericRepository[T]) ListAll(ctx context.Context, filter FilterFunc) ([]T, error) {
	db := r.db.WithContext(ctx)
	if filter != nil {
		db = filter(db)
	}
	var list []T
	if err := db.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// Create 新建实体。e 应为指向实体的指针（&CrudItem{...}），GORM 会回填主键。
func (r *GenericRepository[T]) Create(ctx context.Context, e *T) error {
	return r.db.WithContext(ctx).Create(e).Error
}

// Update 全量保存（Save 更新所有字段，含零值）。
func (r *GenericRepository[T]) Update(ctx context.Context, e *T) error {
	return r.db.WithContext(ctx).Save(e).Error
}

// Delete 软删：GORM 软删 UPDATE 无法经回调注入 deleted_by（见 audit 包注释），
// 故软删实体改用 Updates 双写 deleted_at + deleted_by；GORM 自动补 WHERE deleted_at IS NULL。
// 非软删实体（无 DeletedAt 字段）退回硬删。如需清理关联表，模块自行包装事务。
func (r *GenericRepository[T]) Delete(ctx context.Context, id uint) error {
	if isSoftDelete[T]() {
		return r.db.WithContext(ctx).Model(new(T)).Where("id = ?", id).Updates(softDeleteUpdates(ctx)).Error
	}
	return r.db.WithContext(ctx).Delete(new(T), id).Error
}

// BatchDelete 批量软删（同 Delete 双写逻辑）。空 ids 直接返回，避免 IN () 语法错误。
func (r *GenericRepository[T]) BatchDelete(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	if isSoftDelete[T]() {
		return r.db.WithContext(ctx).Model(new(T)).Where("id IN ?", ids).Updates(softDeleteUpdates(ctx)).Error
	}
	return r.db.WithContext(ctx).Delete(new(T), ids).Error
}
