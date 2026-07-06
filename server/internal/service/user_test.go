package service

import (
	"context"
	"testing"

	"gva/internal/model"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/audit"
	"gva/internal/pkg/datascope"
	"gva/internal/pkg/pagination"
	"gva/internal/repository"
	"gva/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newUserSvc 构造装配好的 UserService（含 admin/user 两个角色）。
func newUserSvc(t *testing.T) *UserService {
	t.Helper()
	db := testutil.NewTestDB(t)
	// 建角色供测试用
	roleRepo := repository.NewRoleRepository(db)
	require.NoError(t, roleRepo.Create(context.Background(), &model.Role{Code: "admin", Name: "管理员", Status: "active"}))
	require.NoError(t, roleRepo.Create(context.Background(), &model.Role{Code: "user", Name: "用户", Status: "active"}))
	return NewUserService(repository.NewUserRepository(db), nil)
}

func TestUserService_Create(t *testing.T) {
	svc := newUserSvc(t)
	u, err := svc.Create(context.Background(), "u1", "张三", "u1@e.com", "13800000000", []string{"admin"}, "active", "pass123")
	require.NoError(t, err)
	assert.NotZero(t, u.ID)
	assert.Equal(t, []string{"admin"}, u.Roles)
}

func TestUserService_Create_DuplicateUsername(t *testing.T) {
	svc := newUserSvc(t)
	ctx := context.Background()
	_, err := svc.Create(ctx, "u1", "n", "", "", []string{"user"}, "active", "pass123")
	require.NoError(t, err)
	_, err = svc.Create(ctx, "u1", "n2", "", "", []string{"user"}, "active", "pass123")
	require.Error(t, err)
	e, ok := apperr.As(err)
	require.True(t, ok)
	assert.Equal(t, 409, e.Status)
}

func TestUserService_Create_UnknownRole(t *testing.T) {
	svc := newUserSvc(t)
	_, err := svc.Create(context.Background(), "u1", "n", "", "", []string{"ghost"}, "active", "pass123")
	require.Error(t, err)
	e, ok := apperr.As(err)
	require.True(t, ok)
	assert.Equal(t, 404, e.Status)
}

func TestUserService_Get_NotFound(t *testing.T) {
	svc := newUserSvc(t)
	_, err := svc.Get(context.Background(), 9999)
	require.Error(t, err)
	e, ok := apperr.As(err)
	require.True(t, ok)
	assert.Equal(t, 404, e.Status)
}

func TestUserService_List(t *testing.T) {
	svc := newUserSvc(t)
	ctx := context.Background()
	// 不同 username 避免唯一约束冲突
	for i := 0; i < 12; i++ {
		username := "u" + string(rune('a'+i))
		_, err := svc.Create(ctx, username, "n", "", "", []string{"user"}, "active", "pass123")
		require.NoError(t, err)
	}
	q := pagination.Query{Page: 1, Size: 10}
	q.Normalize()
	res, err := svc.List(ctx, q, "")
	require.NoError(t, err)
	assert.Equal(t, int64(12), res.Total)
	assert.Len(t, res.Records, 10)
	// 每条 UserInfo 含 roles
	assert.NotEmpty(t, res.Records[0].Roles)
}

func TestUserService_Update_DisableSelf(t *testing.T) {
	svc := newUserSvc(t)
	ctx := context.Background()
	u, err := svc.Create(ctx, "u1", "n", "", "", []string{"admin"}, "active", "pass123")
	require.NoError(t, err)
	inactive := "inactive"
	_, err = svc.Update(ctx, u.ID, u.ID, &UserUpdateReq{Status: &inactive})
	require.Error(t, err)
	e, ok := apperr.As(err)
	require.True(t, ok)
	assert.Equal(t, 409, e.Status)
}

func TestUserService_Delete_Self(t *testing.T) {
	svc := newUserSvc(t)
	ctx := context.Background()
	u, err := svc.Create(ctx, "u1", "n", "", "", []string{"admin"}, "active", "pass123")
	require.NoError(t, err)
	err = svc.Delete(ctx, u.ID, u.ID)
	require.Error(t, err)
	e, ok := apperr.As(err)
	require.True(t, ok)
	assert.Equal(t, 409, e.Status)
}

func TestUserService_Delete_Other(t *testing.T) {
	svc := newUserSvc(t)
	ctx := context.Background()
	u, err := svc.Create(ctx, "u1", "n", "", "", []string{"admin"}, "active", "pass123")
	require.NoError(t, err)
	require.NoError(t, svc.Delete(ctx, u.ID, 88888)) // operator 他人
	_, err = svc.Get(ctx, u.ID)
	assert.Error(t, err) // 软删
}

func TestUserService_Export(t *testing.T) {
	svc := newUserSvc(t)
	ctx := context.Background()
	_, err := svc.Create(ctx, "u1", "张三", "e@e.com", "138", []string{"admin"}, "active", "pass123")
	require.NoError(t, err)
	q := pagination.Query{}
	q.Normalize()
	csv, err := svc.Export(ctx, q, "")
	require.NoError(t, err)
	assert.Contains(t, csv, "username")
	assert.Contains(t, csv, "u1")
}

// TestUserService_List_DataScope 端到端验证 M8 数据范围：
//
//	alice（dept 角色，归属 childA）调用 List 只看到本部门（alice+bob），看不到 childB 的 carol；
//	super（通配权限超管）调用 List 看到全部。
//	scope 经 audit.WithUserID 注入 ctx，UserService 内部解析后传给 repo 过滤。
func TestUserService_List_DataScope(t *testing.T) {
	db := testutil.NewTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Role{}, &model.Permission{}, &model.Dept{}))
	ctx := context.Background()

	deptRepo := repository.NewDeptRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	userRepo := repository.NewUserRepository(db)

	// 部门 childA / childB
	childA := &model.Dept{ParentID: 0, Name: "A", Status: "active"}
	childB := &model.Dept{ParentID: 0, Name: "B", Status: "active"}
	require.NoError(t, deptRepo.Create(ctx, childA))
	require.NoError(t, deptRepo.Create(ctx, childB))

	// 角色：dept（本部门）+ super（通配超管）
	deptRole := &model.Role{Code: "dept", Name: "本部门", Status: "active", DataScope: datascope.ScopeDept}
	superRole := &model.Role{Code: "super", Name: "超管", Status: "active", DataScope: datascope.ScopeSelf}
	require.NoError(t, roleRepo.Create(ctx, deptRole))
	require.NoError(t, roleRepo.Create(ctx, superRole))
	star := &model.Permission{Code: "*", Name: "通配", Type: "api"}
	require.NoError(t, permRepo.Create(ctx, star))
	require.NoError(t, roleRepo.ReplaceRolePermissions(ctx, superRole.ID, []uint{star.ID}))

	// 三个用户：alice/bob 在 childA，carol 在 childB
	mkUser := func(name string, deptID uint, roleID uint) *model.User {
		u := &model.User{Username: name, Password: "x", Status: "active", DeptID: deptID}
		require.NoError(t, userRepo.Create(ctx, u))
		require.NoError(t, userRepo.ReplaceRoles(ctx, u.ID, []uint{roleID}))
		return u
	}
	alice := mkUser("alice", childA.ID, deptRole.ID)
	mkUser("bob", childA.ID, deptRole.ID)
	mkUser("carol", childB.ID, deptRole.ID)
	super := mkUser("super", childB.ID, superRole.ID)

	svc := NewUserService(userRepo, datascope.NewResolver(userRepo, deptRepo))
	q := pagination.Query{Page: 1, Size: 50}
	q.Normalize()

	// alice 视角：仅本部门 childA → alice+bob（2 条），carol 不可见
	aliceCtx := audit.WithUserID(ctx, alice.ID)
	res, err := svc.List(aliceCtx, q, "")
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total)
	names := map[string]bool{}
	for _, u := range res.Records {
		names[u.Username] = true
	}
	assert.True(t, names["alice"])
	assert.True(t, names["bob"])
	assert.False(t, names["carol"])

	// super 视角：通配 → 全部 4 条
	superCtx := audit.WithUserID(ctx, super.ID)
	res, err = svc.List(superCtx, q, "")
	require.NoError(t, err)
	assert.Equal(t, int64(4), res.Total)
}
