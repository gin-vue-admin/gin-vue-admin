package handler

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoginLogHandler_List admin 登录已产生 login_log 记录，分页查询应非空。
func TestLoginLogHandler_List(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/system/login-log?page=1&size=10", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	data, _ := decodeResult(t, w).Data.(map[string]any)
	records, _ := data["records"].([]any)
	assert.NotEmpty(t, records, "admin 登录后 login_log 应有记录")
}

// TestLoginLogHandler_Get 从 List 取首个 id 查详情，应 200。
func TestLoginLogHandler_Get(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/system/login-log?page=1&size=10", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	data, _ := decodeResult(t, w).Data.(map[string]any)
	records, _ := data["records"].([]any)
	require.NotEmpty(t, records, "需先有 login_log 记录")
	first, _ := records[0].(map[string]any)
	id := strconv.Itoa(int(first["id"].(float64)))

	w = authedReq(t, r, http.MethodGet, "/api/system/login-log/"+id, token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	d, _ := decodeResult(t, w).Data.(map[string]any)
	assert.Equal(t, first["id"], d["id"])
}

// TestLoginLogHandler_Clear 清空后再 List 应为空。
func TestLoginLogHandler_Clear(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodDelete, "/api/system/login-log/clear", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())

	w = authedReq(t, r, http.MethodGet, "/api/system/login-log?page=1&size=10", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	data, _ := decodeResult(t, w).Data.(map[string]any)
	records, _ := data["records"].([]any)
	assert.Empty(t, records, "Clear 后 login_log 应为空")
}

// TestOperationLogHandler_List 未挂 OperationLog 中间件，records 应为空数组（证明端点可达）。
func TestOperationLogHandler_List(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/system/operation-log?page=1&size=10", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	data, _ := decodeResult(t, w).Data.(map[string]any)
	records, _ := data["records"].([]any)
	assert.Empty(t, records, "未挂中间件，operation_log 应为空")
}

// TestOperationLogHandler_Clear 清空操作日志，应 200。
func TestOperationLogHandler_Clear(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodDelete, "/api/system/operation-log/clear", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
}

// TestOperationLogHandler_GetNotFound 不存在的 id → 404。
func TestOperationLogHandler_GetNotFound(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/system/operation-log/99999", token, nil)
	assert.Equal(t, 404, w.Code)
}

// TestLoginLogHandler_BatchDelete 批量删除 admin 登录产生的记录，删后 List 为空。
func TestLoginLogHandler_BatchDelete(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/system/login-log?page=1&size=10", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	data, _ := decodeResult(t, w).Data.(map[string]any)
	records, _ := data["records"].([]any)
	require.NotEmpty(t, records, "需先有 login_log 记录")

	ids := make([]string, 0, len(records))
	for _, rec := range records {
		m, _ := rec.(map[string]any)
		ids = append(ids, strconv.Itoa(int(m["id"].(float64))))
	}
	w = authedReq(t, r, http.MethodDelete, "/api/system/login-log", token, batchDeleteReq{IDs: ids})
	require.Equal(t, 200, w.Code, w.Body.String())

	// 删后 List 为空
	w = authedReq(t, r, http.MethodGet, "/api/system/login-log?page=1&size=10", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	data, _ = decodeResult(t, w).Data.(map[string]any)
	records, _ = data["records"].([]any)
	assert.Empty(t, records, "BatchDelete 后 login_log 应为空")
}

// TestOperationLogHandler_NoToken 未鉴权访问 → 401。
func TestOperationLogHandler_NoToken(t *testing.T) {
	r, _ := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/system/operation-log", "", nil)
	assert.Equal(t, 401, w.Code)
}
