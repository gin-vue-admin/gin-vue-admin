package handler

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPermissionHandler_List admin 分页查询权限列表（seed 后非空）。
func TestPermissionHandler_List(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/permission?page=1&size=5", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	data, _ := decodeResult(t, w).Data.(map[string]any)
	records, _ := data["records"].([]any)
	assert.NotEmpty(t, records, "seed 后权限列表非空")
}

// TestPermissionHandler_ListAll all=true 返回全量数组（非分页结构）。
func TestPermissionHandler_ListAll(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/permission?all=true", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	list, _ := decodeResult(t, w).Data.([]any)
	assert.NotEmpty(t, list, "all=true 应返回数组")
}

// TestPermissionHandler_CRUD 创建→查询→更新→删除 主链路。
func TestPermissionHandler_CRUD(t *testing.T) {
	r, token := newAppServer(t)

	// 创建
	body := permissionCreateReq{
		Name: "测试权限", Code: "test:perm", Module: "test",
		Description: "测试", Status: "active",
	}
	w := authedReq(t, r, http.MethodPost, "/api/permission", token, body)
	require.Equalf(t, 200, w.Code, "create body=%s", w.Body.String())
	d, _ := decodeResult(t, w).Data.(map[string]any)
	id := strconv.Itoa(int(d["id"].(float64)))

	// 查询
	w = authedReq(t, r, http.MethodGet, "/api/permission/"+id, token, nil)
	require.Equal(t, 200, w.Code)
	d, _ = decodeResult(t, w).Data.(map[string]any)
	assert.Equal(t, "test:perm", d["code"])

	// 更新
	body.Name = "测试权限改名"
	w = authedReq(t, r, http.MethodPut, "/api/permission/"+id, token, body)
	require.Equal(t, 200, w.Code, w.Body.String())

	// 删除
	w = authedReq(t, r, http.MethodDelete, "/api/permission/"+id, token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())

	// 删后 get → 404
	w = authedReq(t, r, http.MethodGet, "/api/permission/"+id, token, nil)
	assert.Equal(t, 404, w.Code)
}

// TestPermissionHandler_BatchDelete 批量删除。
func TestPermissionHandler_BatchDelete(t *testing.T) {
	r, token := newAppServer(t)
	ids := make([]string, 0, 2)
	for _, code := range []string{"batch:1", "batch:2"} {
		w := authedReq(t, r, http.MethodPost, "/api/permission", token, permissionCreateReq{
			Name: code, Code: code, Module: "batch", Status: "active",
		})
		require.Equal(t, 200, w.Code, w.Body.String())
		d, _ := decodeResult(t, w).Data.(map[string]any)
		ids = append(ids, strconv.Itoa(int(d["id"].(float64))))
	}
	w := authedReq(t, r, http.MethodDelete, "/api/permission", token, batchDeleteReq{IDs: ids})
	require.Equal(t, 200, w.Code, w.Body.String())
}

// TestPermissionHandler_Export 导出 CSV。
func TestPermissionHandler_Export(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/permission/export", token, nil)
	require.Equal(t, 200, w.Code)
	assert.Equal(t, "导出成功", decodeResult(t, w).Msg)
}

// TestPermissionHandler_NoToken 未鉴权 → 401。
func TestPermissionHandler_NoToken(t *testing.T) {
	r, _ := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/permission", "", nil)
	assert.Equal(t, 401, w.Code)
}

// TestPermissionHandler_Validation 缺必填字段 → 422。
func TestPermissionHandler_Validation(t *testing.T) {
	r, token := newAppServer(t)
	// 缺 code/module/status
	w := authedReq(t, r, http.MethodPost, "/api/permission", token, permissionCreateReq{Name: "x"})
	assert.Equal(t, 422, w.Code)
}
