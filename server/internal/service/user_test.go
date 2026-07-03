package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gva/internal/model"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/pagination"
	"gva/internal/repository"
	"gva/internal/testutil"
)

// newUserSvc 构造装配好的 UserService（含 admin/user 两个角色）。
func newUserSvc(t *testing.T) *UserService {
	t.Helper()
	db := testutil.NewTestDB(t)
	// 建角色供测试用
	roleRepo := repository.NewRoleRepository(db)
	require.NoError(t, roleRepo.Create(context.Background(), &model.Role{Code: "admin", Name: "管理员", Status: "active"}))
	require.NoError(t, roleRepo.Create(context.Background(), &model.Role{Code: "user", Name: "用户", Status: "active"}))
	return NewUserService(repository.NewUserRepository(db))
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
