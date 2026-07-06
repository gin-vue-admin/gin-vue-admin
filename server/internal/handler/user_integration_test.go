package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"gva/internal/config"
	"gva/internal/middleware"
	"gva/internal/pkg/async"
	"gva/internal/pkg/audit"
	"gva/internal/pkg/datascope"
	"gva/internal/pkg/jwt"
	"gva/internal/repository"
	"gva/internal/service"
	"gva/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestServer 装配 auth + user 模块的最小 gin 引擎（含 AuthRequired），
// 返回 engine 与 admin 登录 token，供 user CRUD 测试复用。
func newTestServer(t *testing.T) (r *gin.Engine, adminToken string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	require.NoError(t, audit.Register(db))

	userRepo := repository.NewUserRepository(db)
	deptRepo := repository.NewDeptRepository(db)
	loginLogRepo := repository.NewLoginLogRepository(db)
	jwtMgr := jwt.NewManager(config.JWTConfig{
		Secret: "test-secret-very-long-for-hs256-signing", AccessTTL: 3600, RefreshTTL: 86400, Issuer: "gva-test",
	})
	authSvc := service.NewAuthService(userRepo, db, jwtMgr, async.SyncRunner{}, loginLogRepo)
	require.NoError(t, authSvc.Seed(context.Background()))

	userSvc := service.NewUserService(userRepo, datascope.NewResolver(userRepo, deptRepo))
	authH := NewAuthHandler(authSvc)
	userH := NewUserHandler(userSvc)

	r = gin.New()
	api := r.Group("/api")
	api.POST("/auth/sessions", authH.Login)
	user := api.Group("/user")
	user.Use(middleware.AuthRequired(jwtMgr))
	{
		user.GET("", userH.List)
		user.GET("/:id", userH.Get)
		user.POST("", userH.Create)
		user.PUT("/:id", userH.Update)
		user.DELETE("/:id", userH.Delete)
	}

	// admin 登录拿 token
	w := doJSON(t, r, http.MethodPost, "/api/auth/sessions",
		LoginRequest{Username: "admin", Password: "123456"})
	require.Equal(t, 200, w.Code)
	data, _ := decodeResult(t, w).Data.(map[string]any)
	return r, data["accessToken"].(string)
}

// authedReq 带可选 Bearer token 发 JSON 请求。
func authedReq(t *testing.T, r http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// seedUser2 通过 admin 创建一个普通用户，返回其 id（字符串）。
func seedUser2(t *testing.T, r http.Handler, token, username string) string {
	w := authedReq(t, r, http.MethodPost, "/api/user", token,
		service.UserCreateReq{Username: username, RealName: username, Status: "active", Password: "pass1234", Roles: []string{"user"}})
	require.Equalf(t, 200, w.Code, "create user body=%s", w.Body.String())
	d, _ := decodeResult(t, w).Data.(map[string]any)
	return strconv.Itoa(int(d["id"].(float64)))
}

// TestUserHandler_CreateAndList admin 创建用户后列表可见。
func TestUserHandler_CreateAndList(t *testing.T) {
	r, token := newTestServer(t)
	// 初始列表（含 seed 的 admin/user）
	w := authedReq(t, r, http.MethodGet, "/api/user?page=1&size=10", token, nil)
	require.Equal(t, 200, w.Code)
	before, _ := decodeResult(t, w).Data.(map[string]any)
	beforeTotal, _ := before["total"].(float64)

	// 创建新用户
	id := seedUser2(t, r, token, "alice")
	require.NotEmpty(t, id)

	// 列表 total +1
	w = authedReq(t, r, http.MethodGet, "/api/user?page=1&size=10", token, nil)
	require.Equal(t, 200, w.Code)
	after, _ := decodeResult(t, w).Data.(map[string]any)
	afterTotal, _ := after["total"].(float64)
	assert.Equal(t, int64(beforeTotal)+1, int64(afterTotal))
}

// TestUserHandler_GetAndUpdate 详情 + 更新。
func TestUserHandler_GetAndUpdate(t *testing.T) {
	r, token := newTestServer(t)
	id := seedUser2(t, r, token, "bob")

	// 详情
	w := authedReq(t, r, http.MethodGet, "/api/user/"+id, token, nil)
	require.Equal(t, 200, w.Code)
	d, _ := decodeResult(t, w).Data.(map[string]any)
	assert.Equal(t, "bob", d["username"])

	// 更新（停用）
	realName := "鲍勃"
	status := "inactive"
	w = authedReq(t, r, http.MethodPut, "/api/user/"+id, token,
		service.UserUpdateReq{RealName: &realName, Status: &status})
	require.Equalf(t, 200, w.Code, "update body=%s", w.Body.String())
}

// TestUserHandler_Delete 删除他人 → 后续 get 404。
func TestUserHandler_Delete(t *testing.T) {
	r, token := newTestServer(t)
	id := seedUser2(t, r, token, "carol")

	w := authedReq(t, r, http.MethodDelete, "/api/user/"+id, token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())

	// 软删后 get 应 404
	w = authedReq(t, r, http.MethodGet, "/api/user/"+id, token, nil)
	assert.Equal(t, 404, w.Code)
}

// TestUserHandler_NoToken 未鉴权访问 → 401。
func TestUserHandler_NoToken(t *testing.T) {
	r, _ := newTestServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/user", "", nil)
	assert.Equal(t, 401, w.Code)
}
