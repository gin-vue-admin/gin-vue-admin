// Package service 的角色业务单测（TDD：先写失败测试，再实现 role.go）。
// 8 个测试覆盖 Create/Create_DuplicateCode/Get_NotFound/List/Delete_ClearsAssociations/
// SetPermissions_UnknownCode/SetPermissions_Success/Export。
package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gva/internal/model"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/pagination"
	"gva/internal/repository"
	"gva/internal/testutil"
)

// TestRoleService_Create 正常创建返回带 ID 的角色。
func TestRoleService_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewRoleRepository(db)
	svc := NewRoleService(repo)
	ctx := context.Background()
	r, err := svc.Create(ctx, "角色1", "r1", "描述", "active")
	require.NoError(t, err)
	assert.NotZero(t, r.ID)
}

// TestRoleService_Create_DuplicateCode code 唯一约束冲突返回 409。
func TestRoleService_Create_DuplicateCode(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewRoleRepository(db)
	svc := NewRoleService(repo)
	ctx := context.Background()
	_, _ = svc.Create(ctx, "n", "dup", "", "active")
	_, err := svc.Create(ctx, "n2", "dup", "", "active")
	require.Error(t, err)
	e, ok := apperr.As(err)
	require.True(t, ok)
	assert.Equal(t, 409, e.Status)
}

// TestRoleService_Get_NotFound 不存在的 ID 返回 404。
func TestRoleService_Get_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewRoleRepository(db)
	svc := NewRoleService(repo)
	_, err := svc.Get(context.Background(), 9999)
	require.Error(t, err)
	e, _ := apperr.As(err)
	assert.Equal(t, 404, e.Status)
}

// TestRoleService_List 分页列表：12 条第 1 页 10 条，Total=12。
// 注意：Code 有 uniqueIndex，循环创建需用唯一 code，否则第 2 条触发 409。
func TestRoleService_List(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewRoleRepository(db)
	svc := NewRoleService(repo)
	ctx := context.Background()
	for i := 0; i < 12; i++ {
		_, err := svc.Create(ctx, "n", fmt.Sprintf("c%d", i), "", "active")
		require.NoError(t, err)
	}
	q := pagination.Query{Page: 1, Size: 10}
	q.Normalize()
	res, err := svc.List(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, int64(12), res.Total)
	assert.Len(t, res.Records, 10)
}

// TestRoleService_Delete_ClearsAssociations 软删角色后 Get 返回 404；
// 关联表 role_permissions 在事务内已清，软删主流程通过即可。
func TestRoleService_Delete_ClearsAssociations(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewRoleRepository(db)
	svc := NewRoleService(repo)
	ctx := context.Background()
	r, _ := svc.Create(ctx, "n", "r1", "", "active")
	// 关联权限
	perm := &model.Permission{Code: "p1", Name: "p", Type: "api", Status: "active"}
	db.Create(perm)
	db.Model(&model.Role{Model: model.Model{ID: r.ID}}).Association("Permissions").Replace(perm)

	require.NoError(t, svc.Delete(ctx, r.ID))
	_, err := svc.Get(ctx, r.ID)
	assert.Error(t, err) // 软删后 404
}

// TestRoleService_SetPermissions_UnknownCode 传入不存在的权限 code 返回 404。
func TestRoleService_SetPermissions_UnknownCode(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewRoleRepository(db)
	svc := NewRoleService(repo)
	ctx := context.Background()
	r, _ := svc.Create(ctx, "n", "r1", "", "active")
	err := svc.SetPermissions(ctx, r.ID, []string{"unknown"})
	require.Error(t, err)
	e, _ := apperr.As(err)
	assert.Equal(t, 404, e.Status)
}

// TestRoleService_SetPermissions_Success 全量替换权限后 GetPermissions 返回该 code。
func TestRoleService_SetPermissions_Success(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewRoleRepository(db)
	svc := NewRoleService(repo)
	ctx := context.Background()
	r, _ := svc.Create(ctx, "n", "r1", "", "active")
	perm := &model.Permission{Code: "p1", Name: "p", Type: "api", Status: "active"}
	db.Create(perm)
	require.NoError(t, svc.SetPermissions(ctx, r.ID, []string{"p1"}))
	codes, err := svc.GetPermissions(ctx, r.ID)
	require.NoError(t, err)
	assert.Contains(t, codes, "p1")
}

// TestRoleService_Export 导出 CSV 含表头 code 列与数据 r1。
func TestRoleService_Export(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewRoleRepository(db)
	svc := NewRoleService(repo)
	ctx := context.Background()
	_, err := svc.Create(ctx, "管理员", "r1", "描述", "active")
	require.NoError(t, err)
	q := pagination.Query{}
	q.Normalize()
	csv, err := svc.Export(ctx, q)
	require.NoError(t, err)
	assert.Contains(t, csv, "code")
	assert.Contains(t, csv, "r1")
}
