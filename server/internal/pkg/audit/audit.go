// Package audit 提供操作者审计：从请求上下文取当前用户 ID，通过 GORM 回调自动写入
// 实体的 CreatedBy/UpdatedBy 字段。DeletedBy 由 repository 层在软删时显式写入
// （GORM 软删 UPDATE 由 SoftDeleteDeleteClause 控制只 SET deleted_at，回调无法注入额外列）。
//
// 用法：middleware.AuthRequired 解析 token 后调 WithUserID 注入 request context；
// db 初始化时调 Register 注册回调；之后所有 Create/Update 自动带上操作者。
package audit

import (
	"context"
	"fmt"
	"reflect"

	"gorm.io/gorm"
)

// ctxKey 私有上下文键类型，避免与其他包冲突。
type ctxKey struct{}

// WithUserID 把当前用户 ID 注入 context（供 GORM 回调读取）。
func WithUserID(ctx context.Context, userID uint) context.Context {
	return context.WithValue(ctx, ctxKey{}, userID)
}

// UserIDFrom 从 context 取当前用户 ID；不存在返回 ok=false。
func UserIDFrom(ctx context.Context) (uint, bool) {
	uid, ok := ctx.Value(ctxKey{}).(uint)
	return uid, ok
}

// Register 在 db 上注册 GORM 回调：
//   - Create 前：写入 CreatedBy + UpdatedBy
//   - Update 前：写入 UpdatedBy
//
// 字段经反射定位（嵌入 model.Model 的匿名字段会被 FieldByName 自动 promote），
// 故所有嵌入 model.Model 的实体自动受益，无需逐个实现钩子。
func Register(db *gorm.DB) error {
	if err := db.Callback().Create().Before("gorm:create").Register("audit:create", func(tx *gorm.DB) {
		uid, ok := UserIDFrom(tx.Statement.Context)
		if !ok {
			return
		}
		setUintField(tx.Statement.Dest, "CreatedBy", uid)
		setUintField(tx.Statement.Dest, "UpdatedBy", uid)
	}); err != nil {
		return fmt.Errorf("register audit:create callback: %w", err)
	}
	if err := db.Callback().Update().Before("gorm:update").Register("audit:update", func(tx *gorm.DB) {
		uid, ok := UserIDFrom(tx.Statement.Context)
		if !ok {
			return
		}
		setUintField(tx.Statement.Dest, "UpdatedBy", uid)
	}); err != nil {
		return fmt.Errorf("register audit:update callback: %w", err)
	}
	return nil
}

// setUintField 反射定位 dest（struct / 指针 / 切片）中名为 field 的 uint 字段并赋值。
// 支持嵌入匿名字段的 promote（model.Model 内的审计字段）。忽略无此字段的实体。
func setUintField(dest any, field string, val uint) {
	v := reflect.ValueOf(dest)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Struct:
		setStructUint(v, field, val)
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			if elem.Kind() == reflect.Pointer {
				if elem.IsNil() {
					continue
				}
				elem = elem.Elem()
			}
			if elem.Kind() == reflect.Struct {
				setStructUint(elem, field, val)
			}
		}
	}
}

func setStructUint(v reflect.Value, field string, val uint) {
	f := v.FieldByName(field)
	if f.IsValid() && f.CanSet() && f.Kind() == reflect.Uint {
		f.SetUint(uint64(val))
	}
}
