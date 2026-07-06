package repository

import (
	"context"
	"testing"

	"gva/internal/model"
	"gva/internal/pkg/datascope"
	"gva/internal/pkg/pagination"
	"gva/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserRepo_CRUD 覆盖 Create/FindByID/Update 主流程。
func TestUserRepo_CRUD(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()
	u := &model.User{Username: "u1", Password: "x", RealName: "张三", Status: "active"}
	require.NoError(t, repo.Create(ctx, u))
	assert.NotZero(t, u.ID)

	got, err := repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "u1", got.Username)

	got.RealName = "李四"
	require.NoError(t, repo.Update(ctx, got))
	got2, _ := repo.FindByID(ctx, u.ID)
	assert.Equal(t, "李四", got2.RealName)
}

// TestUserRepo_List_FilterRole 验证 List 按 roleCode 过滤：仅关联 admin 角色的用户被返回。
func TestUserRepo_List_FilterRole(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewUserRepository(db)
	roleRepo := NewRoleRepository(db)
	ctx := context.Background()
	// 建角色 admin
	role := &model.Role{Code: "admin", Name: "管理员", Status: "active"}
	roleRepo.Create(ctx, role)
	// 建 2 个用户，1 个关联 admin，1 个无角色
	u1 := &model.User{Username: "u1", Password: "x", Status: "active"}
	repo.Create(ctx, u1)
	repo.ReplaceRoles(ctx, u1.ID, []uint{role.ID})
	u2 := &model.User{Username: "u2", Password: "x", Status: "active"}
	repo.Create(ctx, u2)

	q := pagination.Query{Page: 1, Size: 10}
	q.Normalize()
	users, total, err := repo.List(ctx, q, "admin", datascope.Scope{All: true})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total) // 仅 u1 有 admin 角色
	assert.Len(t, users, 1)
	assert.Equal(t, "u1", users[0].Username)
}

// TestUserRepo_Delete_ClearsUserRoles 验证删除用户时事务内清除 user_roles 关联。
func TestUserRepo_Delete_ClearsUserRoles(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewUserRepository(db)
	roleRepo := NewRoleRepository(db)
	ctx := context.Background()
	role := &model.Role{Code: "r", Name: "r", Status: "active"}
	roleRepo.Create(ctx, role)
	u := &model.User{Username: "u1", Password: "x", Status: "active"}
	repo.Create(ctx, u)
	repo.ReplaceRoles(ctx, u.ID, []uint{role.ID})

	require.NoError(t, repo.Delete(ctx, u.ID))
	_, err := repo.FindByID(ctx, u.ID)
	assert.Error(t, err) // 软删
	var cnt int64
	db.Table("user_roles").Where("user_id = ?", u.ID).Count(&cnt)
	assert.Equal(t, int64(0), cnt) // user_roles 清除
}

// TestUserRepo_ReplaceRoles_And_FindRoleIDsByCodes 覆盖角色替换 + 按 code 反查 id 映射。
func TestUserRepo_ReplaceRoles_And_FindRoleIDsByCodes(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewUserRepository(db)
	roleRepo := NewRoleRepository(db)
	ctx := context.Background()
	r1 := &model.Role{Code: "admin", Name: "a", Status: "active"}
	r2 := &model.Role{Code: "user", Name: "u", Status: "active"}
	roleRepo.Create(ctx, r1)
	roleRepo.Create(ctx, r2)
	u := &model.User{Username: "u1", Password: "x", Status: "active"}
	repo.Create(ctx, u)

	require.NoError(t, repo.ReplaceRoles(ctx, u.ID, []uint{r1.ID, r2.ID}))
	m, err := repo.FindRoleIDsByCodes(ctx, []string{"admin", "user", "unknown"})
	require.NoError(t, err)
	assert.Contains(t, m, "admin")
	assert.Contains(t, m, "user")
	assert.NotContains(t, m, "unknown")
}
