package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gva/internal/pkg/apperr"
)

// fakeReader 测试用 PermissionReader 桩，记录调用次数。
type fakeReader struct {
	codes []string
	calls int
}

func (f *fakeReader) GetUserPermissionCodes(ctx context.Context, userID uint) ([]string, error) {
	f.calls++
	return f.codes, nil
}

// setupRouter 构造测试路由：注入 userID=1（uint），挂载 RequirePermission。
// 开头调 InvalidateAll 清全局 permCache，避免测试间互相污染。
func setupRouter(reader PermissionReader, codes ...string) *gin.Engine {
	InvalidateAll() // 确保每个测试从空缓存开始，隔离副作用
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(ContextKeyUserID, uint(1)); c.Next() })
	r.GET("/x", RequirePermission(reader, codes...), func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	return r
}

func TestRequirePermission_Allow(t *testing.T) {
	r := setupRouter(&fakeReader{codes: []string{"permission:list"}}, "permission:list")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func TestRequirePermission_Deny(t *testing.T) {
	r := setupRouter(&fakeReader{codes: []string{"other"}}, "permission:list")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 403, w.Code)
}

func TestRequirePermission_SuperAdmin(t *testing.T) {
	r := setupRouter(&fakeReader{codes: []string{"*"}}, "permission:list")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func TestRequirePermission_Cache(t *testing.T) {
	reader := &fakeReader{codes: []string{"permission:list"}}
	r := setupRouter(reader, "permission:list")
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/x", nil)
		r.ServeHTTP(w, req)
	}
	assert.Equal(t, 1, reader.calls) // 缓存命中，只查一次 DB
	InvalidateAll()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 2, reader.calls) // 失效后重查
}

func TestPermissionReader_Interface(t *testing.T) {
	// 确认 apperr.Forbidden 返回 403
	e := apperr.Forbidden("禁止访问")
	assert.Equal(t, 403, e.Status)
}
