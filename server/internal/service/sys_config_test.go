package service

import (
	"context"
	"testing"

	"gva/internal/model"
	"gva/internal/repository"
	"gva/internal/testutil"

	"github.com/stretchr/testify/require"
)

// newSysConfigSvc 构造基于 sqlite 的 SysConfigService，并迁移 sys_config 表。
func newSysConfigSvc(t *testing.T) *SysConfigService {
	t.Helper()
	db := testutil.NewTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.SysConfig{}))
	return NewSysConfigService(repository.NewSysConfigRepository(db))
}

func TestSysConfigService_GetValueAndTypeParsers(t *testing.T) {
	svc := newSysConfigSvc(t)
	ctx := context.Background()

	require.NoError(t, svc.Create(ctx, &model.SysConfig{ConfigKey: "k_str", ConfigValue: "hello", Type: "string"}))
	require.NoError(t, svc.Create(ctx, &model.SysConfig{ConfigKey: "k_bool", ConfigValue: "true", Type: "bool"}))
	require.NoError(t, svc.Create(ctx, &model.SysConfig{ConfigKey: "k_int", ConfigValue: "42", Type: "int"}))
	require.NoError(t, svc.LoadAll(ctx)) // 显式预热（main.go 启动路径）

	require.Equal(t, "hello", svc.GetValue("k_str"))
	require.True(t, svc.GetBool("k_bool"))
	require.False(t, svc.GetBool("k_str")) // 非布尔值 → false
	require.Equal(t, 42, svc.GetInt("k_int", 0))
	require.Equal(t, 9, svc.GetInt("k_missing", 9)) // 不存在 → fallback
	require.Equal(t, 9, svc.GetInt("k_str", 9))     // 解析失败 → fallback
	require.Equal(t, "", svc.GetValue("missing"))
}

func TestSysConfigService_CreateRefreshesCache(t *testing.T) {
	svc := newSysConfigSvc(t)
	ctx := context.Background()
	require.NoError(t, svc.LoadAll(ctx))
	require.Equal(t, "", svc.GetValue("newkey"))

	// Create 后内部 reload，缓存立即可见
	require.NoError(t, svc.Create(ctx, &model.SysConfig{ConfigKey: "newkey", ConfigValue: "newval", Type: "string"}))
	require.Equal(t, "newval", svc.GetValue("newkey"))
}

func TestSysConfigService_SetRefreshesCache(t *testing.T) {
	svc := newSysConfigSvc(t)
	ctx := context.Background()
	require.NoError(t, svc.Create(ctx, &model.SysConfig{ConfigKey: "k", ConfigValue: "v1", Type: "string"}))
	require.NoError(t, svc.LoadAll(ctx))
	require.Equal(t, "v1", svc.GetValue("k"))

	// Set 改值后缓存同步
	require.NoError(t, svc.Set(ctx, "k", "v2"))
	require.Equal(t, "v2", svc.GetValue("k"))

	// CRUD Update 同样刷新缓存
	e, err := svc.GetByKey(ctx, "k")
	require.NoError(t, err)
	e.ConfigValue = "v3"
	require.NoError(t, svc.Update(ctx, e))
	require.Equal(t, "v3", svc.GetValue("k"))
}

func TestSysConfigService_SetMissingKey(t *testing.T) {
	svc := newSysConfigSvc(t)
	err := svc.Set(context.Background(), "no_such_key", "v")
	require.Error(t, err) // 不存在 key 应报错（NotFound 翻译）
}
