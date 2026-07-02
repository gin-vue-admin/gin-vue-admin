// Package service 的单测。本任务仅 Seed，其余业务测试在 Task 5 补全。
package service

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gva/internal/config"
	"gva/internal/model"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/async"
	"gva/internal/pkg/jwt"
	"gva/internal/repository"
)

// newTestDB 用 SQLite 建表，隔离真实 MySQL。
// 每个测试用独立临时文件库，彻底隔离，避免 cache=shared 共享内存库导致测试串数据。
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "test.db")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, model.AutoMigrate(db))
	return db
}

func newAuthSvc(t *testing.T) (*AuthService, *gorm.DB) {
	t.Helper()
	db := newTestDB(t)
	repo := repository.NewUserRepository(db)
	jwtMgr := jwt.NewManager(config.JWTConfig{
		Secret: "test-secret", AccessTTL: 3600, RefreshTTL: 604800, Issuer: "gva-test",
	})
	return NewAuthService(repo, db, jwtMgr, async.SyncRunner{}), db
}

// TestSeed_Idempotent 验证 Seed 幂等：调两次不报错、不重复，
// 最终库里有 2 用户、2 角色、2 权限。
func TestSeed_Idempotent(t *testing.T) {
	svc, db := newAuthSvc(t)
	ctx := context.Background()

	require.NoError(t, svc.Seed(ctx))
	require.NoError(t, svc.Seed(ctx)) // 第二次不报错、不重复

	// 用户
	var users []model.User
	db.Find(&users)
	assert.Len(t, users, 2)
	// 角色
	var roles []model.Role
	db.Find(&roles)
	assert.Len(t, roles, 2)
	// 权限
	var perms []model.Permission
	db.Find(&perms)
	assert.Len(t, perms, 2)
}

// setupSeeded 构造一个已 Seed 过的 AuthService，供 Login 测试复用。
func setupSeeded(t *testing.T) *AuthService {
	t.Helper()
	svc, _ := newAuthSvc(t)
	require.NoError(t, svc.Seed(context.Background()))
	return svc
}

// TestLogin_Success 正确凭据应签发 access/refresh token，ExpiresIn=3600。
func TestLogin_Success(t *testing.T) {
	svc := setupSeeded(t)
	res, err := svc.Login(context.Background(), "admin", "123456")
	require.NoError(t, err)
	assert.NotEmpty(t, res.AccessToken)
	assert.NotEmpty(t, res.RefreshToken)
	assert.Equal(t, 3600, res.ExpiresIn)
}

// TestLogin_WrongPassword 密码错返回 401，文案为"用户名或密码错误"。
func TestLogin_WrongPassword(t *testing.T) {
	svc := setupSeeded(t)
	_, err := svc.Login(context.Background(), "admin", "wrong")
	require.Error(t, err)
	e, ok := apperr.As(err)
	require.True(t, ok)
	assert.Equal(t, 401, e.Status)
	assert.Contains(t, e.Detail, "用户名或密码错误")
}

// TestLogin_UserNotFound 用户不存在时返回与密码错同一文案，防用户枚举。
func TestLogin_UserNotFound(t *testing.T) {
	svc := setupSeeded(t)
	_, err := svc.Login(context.Background(), "nobody", "123456")
	require.Error(t, err)
	// 防枚举：与密码错同一文案
	e, _ := apperr.As(err)
	assert.Contains(t, e.Detail, "用户名或密码错误")
}

// TestLogin_DisabledUser 账户状态非 active 时拒绝登录，返回 401。
func TestLogin_DisabledUser(t *testing.T) {
	svc, db := newAuthSvc(t)
	require.NoError(t, svc.Seed(context.Background()))
	// 禁用 admin
	db.Model(&model.User{}).Where("username = ?", "admin").Update("status", "disabled")
	_, err := svc.Login(context.Background(), "admin", "123456")
	require.Error(t, err)
	e, _ := apperr.As(err)
	assert.Equal(t, 401, e.Status)
}

// TestRefresh_Success 用 refresh token 换发新 token 对，ExpiresIn=3600。
func TestRefresh_Success(t *testing.T) {
	svc := setupSeeded(t)
	// 先登录拿 refresh
	res, err := svc.Login(context.Background(), "admin", "123456")
	require.NoError(t, err)
	// 刷新
	res2, err := svc.Refresh(context.Background(), res.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, res2.AccessToken)
	assert.NotEmpty(t, res2.RefreshToken)
	assert.Equal(t, 3600, res2.ExpiresIn)
}

// TestRefresh_WithAccessTokenFails 用 access token 冒充 refresh 应被拒（类型校验）。
func TestRefresh_WithAccessTokenFails(t *testing.T) {
	svc := setupSeeded(t)
	res, err := svc.Login(context.Background(), "admin", "123456")
	require.NoError(t, err)
	// 用 access token 去刷新应失败
	_, err = svc.Refresh(context.Background(), res.AccessToken)
	require.Error(t, err)
	e, _ := apperr.As(err)
	assert.Equal(t, 401, e.Status)
}

// TestRefresh_InvalidTokenFails 非法 token 字符串刷新失败。
func TestRefresh_InvalidTokenFails(t *testing.T) {
	svc := setupSeeded(t)
	_, err := svc.Refresh(context.Background(), "garbage")
	require.Error(t, err)
}

// TestGetProfile_NormalUser 普通用户 profile：角色含 user，权限含 user:read。
func TestGetProfile_NormalUser(t *testing.T) {
	svc := setupSeeded(t)
	// user 账户登录后取 id
	res, _ := svc.Login(context.Background(), "user", "123456")
	// 从 access token 解析 uid
	claims, err := svc.jwtMgr.Parse(res.AccessToken)
	require.NoError(t, err)
	prof, err := svc.GetProfile(context.Background(), claims.UserID)
	require.NoError(t, err)
	assert.Equal(t, "user", prof.Username)
	assert.Contains(t, prof.Roles, "user")
	assert.Contains(t, prof.Permissions, "user:read")
}

// TestGetProfile_SuperAdminWildcard 超管 profile：权限短路为 ["*"]，角色含 super_admin。
// 关键：判断依据是权限码 "*" 而非角色名，超管语义在权限码层短路。
func TestGetProfile_SuperAdminWildcard(t *testing.T) {
	svc := setupSeeded(t)
	res, _ := svc.Login(context.Background(), "admin", "123456")
	claims, _ := svc.jwtMgr.Parse(res.AccessToken)
	prof, err := svc.GetProfile(context.Background(), claims.UserID)
	require.NoError(t, err)
	assert.Equal(t, []string{"*"}, prof.Permissions)
	assert.Contains(t, prof.Roles, "super_admin")
}

// TestLogout_NoOp 纯 JWT 模式下 Logout 为空操作，恒返回 nil。
func TestLogout_NoOp(t *testing.T) {
	svc := setupSeeded(t)
	assert.NoError(t, svc.Logout(context.Background()))
}
