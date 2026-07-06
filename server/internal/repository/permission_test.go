package repository

import (
	"context"
	"fmt"
	"testing"

	"gva/internal/model"
	"gva/internal/pkg/pagination"
	"gva/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPermissionRepo_CRUD 覆盖 Create/FindByID/Update/Delete（软删）主路径。
func TestPermissionRepo_CRUD(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	// Create
	p := &model.Permission{Code: "test:list", Name: "测试", Type: "api", Module: "test", Status: "active"}
	require.NoError(t, repo.Create(ctx, p))
	assert.NotZero(t, p.ID)

	// FindByID
	got, err := repo.FindByID(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, "test:list", got.Code)

	// FindByID 不存在
	_, err = repo.FindByID(ctx, 9999)
	assert.Error(t, err) // gorm.ErrRecordNotFound

	// Update
	got.Name = "改"
	require.NoError(t, repo.Update(ctx, got))
	got2, _ := repo.FindByID(ctx, p.ID)
	assert.Equal(t, "改", got2.Name)

	// Delete（软删）
	require.NoError(t, repo.Delete(ctx, p.ID))
	_, err = repo.FindByID(ctx, p.ID)
	assert.Error(t, err) // 软删后查不到
}

// TestPermissionRepo_List 验证分页 + module 过滤 + total 计数。
func TestPermissionRepo_List(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewPermissionRepository(db)
	ctx := context.Background()
	for i := 0; i < 15; i++ {
		// Code 有 uniqueIndex，每条用唯一 code 避免 UNIQUE constraint failed。
		require.NoError(t, repo.Create(ctx, &model.Permission{Code: fmt.Sprintf("c:%d", i), Name: "n", Type: "api", Module: "user", Status: "active"}))
	}
	q := pagination.Query{Page: 1, Size: 10}
	q.Normalize()
	perms, total, err := repo.List(ctx, q, "user")
	require.NoError(t, err)
	assert.Equal(t, int64(15), total)
	assert.Len(t, perms, 10)
}

// TestPermissionRepo_ListAll 验证不分页返回全量。
func TestPermissionRepo_ListAll(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewPermissionRepository(db)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(ctx, &model.Permission{Code: fmt.Sprintf("c:%d", i), Name: "n", Type: "api", Module: "user", Status: "active"}))
	}
	q := pagination.Query{}
	q.Normalize()
	perms, err := repo.ListAll(ctx, q, "user")
	require.NoError(t, err)
	assert.Len(t, perms, 3)
}

// TestPermissionRepo_BatchDelete 验证批量软删。
func TestPermissionRepo_BatchDelete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewPermissionRepository(db)
	ctx := context.Background()
	var ids []uint
	for i := 0; i < 3; i++ {
		p := &model.Permission{Code: fmt.Sprintf("c:%d", i), Name: "n", Type: "api", Status: "active"}
		require.NoError(t, repo.Create(ctx, p))
		ids = append(ids, p.ID)
	}
	require.NoError(t, repo.BatchDelete(ctx, ids))
	for _, id := range ids {
		_, err := repo.FindByID(ctx, id)
		assert.Error(t, err)
	}
}

// TestPermissionRepo_GetUserPermissionCodes 验证跨表 Raw SQL：
// user → user_roles → role_permissions → permissions 能取回正确 code 集合。
// 此为 M3.1 权限中间件的关键依赖，需确认标准 JOIN 在 SQLite 正常工作。
func TestPermissionRepo_GetUserPermissionCodes(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewPermissionRepository(db)
	ctx := context.Background()

	// 建权限
	perm := &model.Permission{Code: "test:x", Name: "x", Type: "api", Status: "active"}
	require.NoError(t, repo.Create(ctx, perm))

	// 建角色并关联权限
	role := model.Role{Code: "r1", Name: "r1", Status: "active"}
	require.NoError(t, db.Create(&role).Error)
	require.NoError(t, db.Model(&role).Association("Permissions").Replace(perm))

	// 建用户并关联角色
	user := model.User{Username: "u1", Password: "x", Status: "active"}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Model(&user).Association("Roles").Replace(&role))

	codes, err := repo.GetUserPermissionCodes(ctx, user.ID)
	require.NoError(t, err)
	assert.Contains(t, codes, "test:x")
}
