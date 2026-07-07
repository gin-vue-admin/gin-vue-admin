// Package gormx 提供 gorm/DB 错误的公共判断与转换，消除跨层重复的错误处理样板。
package gormx

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// IsDuplicateKey 判断是否唯一约束冲突。
// errors.Is(gorm.ErrDuplicatedKey) 依赖 driver 翻译错误，部分 driver 版本不触发；
// 故对 SQLite（"UNIQUE constraint failed"）与 MySQL（"Duplicate entry"）额外字符串兜底，
// 兼容测试（SQLite）与生产（MySQL）两种驱动。
func IsDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, gorm.ErrDuplicatedKey) ||
		strings.Contains(err.Error(), "Duplicate entry") ||
		strings.Contains(err.Error(), "UNIQUE constraint failed")
}
