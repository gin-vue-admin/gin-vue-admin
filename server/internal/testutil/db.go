// Package testutil 提供测试公共辅助。NewTestDB 用 SQLite 临时文件库建表，隔离真实 MySQL。
package testutil

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gva/internal/model"
)

// NewTestDB 创建独立的 SQLite 临时文件库并 AutoMigrate。
// 每个测试用独立临时目录，避免 cache=shared 共享内存库导致测试串数据。
func NewTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "test.db")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, model.AutoMigrate(db))
	return db
}
