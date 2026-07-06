package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gva/internal/config"
	"gva/internal/middleware"
	"gva/internal/pkg/async"
	"gva/internal/pkg/audit"
	"gva/internal/pkg/jwt"
	"gva/internal/pkg/response"
	"gva/internal/repository"
	"gva/internal/service"
	"gva/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newAuthTestServer 装配一个最小 gin 引擎，挂 auth 路由 + 必要中间件，
// 用 sqlite + SyncRunner 隔离跑通 handler→service→repository 全链路。
func newAuthTestServer(t *testing.T) (*gin.Engine, *service.AuthService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	require.NoError(t, audit.Register(db))

	userRepo := repository.NewUserRepository(db)
	loginLogRepo := repository.NewLoginLogRepository(db)
	jwtMgr := jwt.NewManager(config.JWTConfig{
		Secret: "test-secret-very-long-for-hs256-signing", AccessTTL: 3600, RefreshTTL: 86400, Issuer: "gva-test",
	})
	authSvc := service.NewAuthService(userRepo, db, jwtMgr, async.SyncRunner{}, loginLogRepo)
	require.NoError(t, authSvc.Seed(context.Background()))

	h := NewAuthHandler(authSvc)
	r := gin.New()
	auth := r.Group("/api/auth")
	{
		auth.POST("/sessions", h.Login)
		auth.POST("/tokens/refresh", h.Refresh)
		auth.DELETE("/sessions", h.Logout)
		auth.GET("/users/me", middleware.AuthRequired(jwtMgr), h.Me)
	}
	return r, authSvc
}

// doJSON 发起 JSON 请求并返回响应体。
func doJSON(t *testing.T, r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// decodeResult 解 ApiResult，返回 data 与原始响应。
func decodeResult(t *testing.T, w *httptest.ResponseRecorder) response.ApiResult {
	t.Helper()
	var res response.ApiResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res), "body=%s", w.Body.String())
	return res
}

// TestAuthHandler_Login_Success admin 登录成功，返回双 token。
func TestAuthHandler_Login_Success(t *testing.T) {
	r, _ := newAuthTestServer(t)
	w := doJSON(t, r, http.MethodPost, "/api/auth/sessions",
		LoginRequest{Username: "admin", Password: "123456"})
	require.Equal(t, 200, w.Code)
	res := decodeResult(t, w)
	assert.Equal(t, 0, res.Code)
	data, _ := res.Data.(map[string]any)
	assert.NotEmpty(t, data["accessToken"])
	assert.NotEmpty(t, data["refreshToken"])
}

// TestAuthHandler_Login_WrongPassword 密码错→401，文案防枚举（不泄露用户是否存在）。
func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	r, _ := newAuthTestServer(t)
	w := doJSON(t, r, http.MethodPost, "/api/auth/sessions",
		LoginRequest{Username: "admin", Password: "wrong-pass"})
	assert.Equal(t, 401, w.Code)
}

// TestAuthHandler_Login_Validation 短密码→422 校验失败，不进 service。
func TestAuthHandler_Login_Validation(t *testing.T) {
	r, _ := newAuthTestServer(t)
	w := doJSON(t, r, http.MethodPost, "/api/auth/sessions",
		LoginRequest{Username: "admin", Password: "123"}) // min=6
	assert.Equal(t, 422, w.Code)
}

// TestAuthHandler_Login_NonexistentUser 不存在的用户同样 401（防枚举，与密码错不可区分）。
func TestAuthHandler_Login_NonexistentUser(t *testing.T) {
	r, _ := newAuthTestServer(t)
	w := doJSON(t, r, http.MethodPost, "/api/auth/sessions",
		LoginRequest{Username: "ghost", Password: "123456"})
	assert.Equal(t, 401, w.Code)
}

// TestAuthHandler_Me_WithToken 登录拿 token 后调 /me，返回当前用户档案。
func TestAuthHandler_Me_WithToken(t *testing.T) {
	r, _ := newAuthTestServer(t)
	// 登录拿 token
	w := doJSON(t, r, http.MethodPost, "/api/auth/sessions",
		LoginRequest{Username: "admin", Password: "123456"})
	require.Equal(t, 200, w.Code)
	data, _ := decodeResult(t, w).Data.(map[string]any)
	token, _ := data["accessToken"].(string)
	require.NotEmpty(t, token)

	// 带 token 调 me
	req := httptest.NewRequest(http.MethodGet, "/api/auth/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code, w.Body.String())
	res := decodeResult(t, w)
	profile, _ := res.Data.(map[string]any)
	assert.Equal(t, "admin", profile["username"])
}

// TestAuthHandler_Me_NoToken 未携带 token→401。
func TestAuthHandler_Me_NoToken(t *testing.T) {
	r, _ := newAuthTestServer(t)
	w := doJSON(t, r, http.MethodGet, "/api/auth/users/me", nil)
	assert.Equal(t, 401, w.Code)
}

// TestAuthHandler_Refresh 登录后用 refreshToken 换新 access token。
func TestAuthHandler_Refresh(t *testing.T) {
	r, _ := newAuthTestServer(t)
	w := doJSON(t, r, http.MethodPost, "/api/auth/sessions",
		LoginRequest{Username: "admin", Password: "123456"})
	require.Equal(t, 200, w.Code)
	data, _ := decodeResult(t, w).Data.(map[string]any)
	refresh, _ := data["refreshToken"].(string)
	require.NotEmpty(t, refresh)

	w = doJSON(t, r, http.MethodPost, "/api/auth/tokens/refresh",
		RefreshRequest{RefreshToken: refresh})
	require.Equal(t, 200, w.Code, w.Body.String())
	res := decodeResult(t, w)
	newData, _ := res.Data.(map[string]any)
	assert.NotEmpty(t, newData["accessToken"])
}
