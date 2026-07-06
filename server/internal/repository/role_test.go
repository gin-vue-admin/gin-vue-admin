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

func newRoleRepo(t *testing.T) RoleRepository {
	return NewRoleRepository(testutil.NewTestDB(t))
}

func TestRoleRepo_CRUD(t *testing.T) {
	repo := newRoleRepo(t)
	ctx := context.Background()
	r := &model.Role{Code: "r1", Name: "角色1", Description: "d", Status: "active"}
	require.NoError(t, repo.Create(ctx, r))
	assert.NotZero(t, r.ID)

	got, err := repo.FindByID(ctx, r.ID)
	require.NoError(t, err)
	assert.Equal(t, "r1", got.Code)

	got.Name = "改"
	require.NoError(t, repo.Update(ctx, got))

	_, err = repo.FindByID(ctx, 9999)
	assert.Error(t, err) // NotFound
}

func TestRoleRepo_List(t *testing.T) {
	repo := newRoleRepo(t)
	ctx := context.Background()
	for i := 0; i < 15; i++ {
		// code 字段为 uniqueIndex，每条必须唯一，否则 UNIQUE constraint failed。
		require.NoError(t, repo.Create(ctx, &model.Role{
			Code: fmt.Sprintf("c%d", i), Name: "n", Status: "active",
		}))
	}
	q := pagination.Query{Page: 1, Size: 10}
	q.Normalize()
	roles, total, err := repo.List(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, int64(15), total)
	assert.Len(t, roles, 10)
}

func TestRoleRepo_Delete_ClearsAssociations(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()
	// 建角色 + 权限 + 关联 + user→role 关联
	role := &model.Role{Code: "r1", Name: "n", Status: "active"}
	repo.Create(ctx, role)
	perm := &model.Permission{Code: "p1", Name: "p", Type: "api", Status: "active"}
	db.Create(perm)
	db.Model(role).Association("Permissions").Replace(perm)
	user := &model.User{Username: "u1", Password: "x", Status: "active"}
	db.Create(user)
	db.Model(user).Association("Roles").Replace(role)

	require.NoError(t, repo.Delete(ctx, role.ID))

	// role 软删
	_, err := repo.FindByID(ctx, role.ID)
	assert.Error(t, err)
	// role_permissions 已清
	var rpCount int64
	db.Table("role_permissions").Where("role_id = ?", role.ID).Count(&rpCount)
	assert.Equal(t, int64(0), rpCount)
	// user_roles 已清
	var urCount int64
	db.Table("user_roles").Where("role_id = ?", role.ID).Count(&urCount)
	assert.Equal(t, int64(0), urCount)
}

func TestRoleRepo_PermissionCodesAndReplace(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()
	role := &model.Role{Code: "r1", Name: "n", Status: "active"}
	repo.Create(ctx, role)
	p1 := &model.Permission{Code: "p1", Name: "p", Type: "api", Status: "active"}
	p2 := &model.Permission{Code: "p2", Name: "p", Type: "api", Status: "active"}
	db.Create(p1)
	db.Create(p2)

	// Replace
	require.NoError(t, repo.ReplaceRolePermissions(ctx, role.ID, []uint{p1.ID, p2.ID}))
	codes, err := repo.GetRolePermissionCodes(ctx, role.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"p1", "p2"}, codes)

	// FindPermissionIDsByCodes
	m, err := repo.FindPermissionIDsByCodes(ctx, []string{"p1", "p2", "unknown"})
	require.NoError(t, err)
	assert.Contains(t, m, "p1")
	assert.Contains(t, m, "p2")
	assert.NotContains(t, m, "unknown") // 未知不在 map
}
