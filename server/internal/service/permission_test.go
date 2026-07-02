// Package service 的权限业务单测（TDD：先写失败测试，再实现 permission.go）。
// 7 个测试覆盖 Create/Create_DuplicateCode/Get_NotFound/List/ListAll/SoftDelete/Export。
package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/pagination"
	"gva/internal/repository"
	"gva/internal/testutil"
)

// TestPermService_Create 正常创建返回带 ID 的权限。
func TestPermService_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewPermissionRepository(db)
	svc := NewPermissionService(repo)
	ctx := context.Background()

	p, err := svc.Create(ctx, "新权限", "new:list", "user", "测试", "active")
	require.NoError(t, err)
	assert.NotZero(t, p.ID)
}

// TestPermService_Create_DuplicateCode code 唯一约束冲突返回 409。
func TestPermService_Create_DuplicateCode(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewPermissionRepository(db)
	svc := NewPermissionService(repo)
	ctx := context.Background()
	_, _ = svc.Create(ctx, "n", "dup", "user", "", "active")
	_, err := svc.Create(ctx, "n2", "dup", "user", "", "active")
	require.Error(t, err)
	e, ok := apperr.As(err)
	require.True(t, ok)
	assert.Equal(t, 409, e.Status)
}

// TestPermService_Get_NotFound 不存在的 ID 返回 404。
func TestPermService_Get_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewPermissionRepository(db)
	svc := NewPermissionService(repo)
	_, err := svc.Get(context.Background(), 9999)
	require.Error(t, err)
	e, ok := apperr.As(err)
	require.True(t, ok)
	assert.Equal(t, 404, e.Status)
}

// TestPermService_List 分页列表：12 条第 1 页 10 条，Total=12。
// 注意：Code 有 uniqueIndex，循环创建需用唯一 code，否则第 2 条触发 409。
func TestPermService_List(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewPermissionRepository(db)
	svc := NewPermissionService(repo)
	ctx := context.Background()
	for i := 0; i < 12; i++ {
		_, err := svc.Create(ctx, "n", fmt.Sprintf("c%d", i), "user", "", "active")
		require.NoError(t, err)
	}
	q := pagination.Query{Page: 1, Size: 10}
	q.Normalize()
	res, err := svc.List(ctx, q, "user")
	require.NoError(t, err)
	assert.Equal(t, int64(12), res.Total)
	assert.Len(t, res.Records, 10)
}

// TestPermService_ListAll 全量列表：5 条全返回。
func TestPermService_ListAll(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewPermissionRepository(db)
	svc := NewPermissionService(repo)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, err := svc.Create(ctx, "n", fmt.Sprintf("c%d", i), "user", "", "active")
		require.NoError(t, err)
	}
	q := pagination.Query{}
	q.Normalize()
	all, err := svc.ListAll(ctx, q, "user")
	require.NoError(t, err)
	assert.Len(t, all, 5)
}

// TestPermService_SoftDelete 软删后 Get 返回 404。
func TestPermService_SoftDelete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewPermissionRepository(db)
	svc := NewPermissionService(repo)
	ctx := context.Background()
	p, _ := svc.Create(ctx, "n", "c", "user", "", "active")
	require.NoError(t, svc.Delete(ctx, p.ID))
	_, err := svc.Get(ctx, p.ID)
	assert.Error(t, err) // 软删后 404
}

// TestPermService_Export 导出 CSV 含表头 code 列与数据 user:list。
func TestPermService_Export(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewPermissionRepository(db)
	svc := NewPermissionService(repo)
	ctx := context.Background()
	_, err := svc.Create(ctx, "用户列表", "user:list", "user", "查看", "active")
	require.NoError(t, err)
	q := pagination.Query{}
	q.Normalize()
	csv, err := svc.Export(ctx, q, "user")
	require.NoError(t, err)
	assert.Contains(t, csv, "code")
	assert.Contains(t, csv, "user:list")
}
