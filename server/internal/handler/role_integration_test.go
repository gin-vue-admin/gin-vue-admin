package handler

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRoleHandler_List admin 分页查询角色列表（seed 后含 super_admin/user 两角色）。
func TestRoleHandler_List(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/role?page=1&size=10", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	data, _ := decodeResult(t, w).Data.(map[string]any)
	records, _ := data["records"].([]any)
	assert.GreaterOrEqual(t, len(records), 2, "seed 后至少有 super_admin/user 两角色")
}

// TestRoleHandler_Get 详情：从列表取首个 id 再查。
func TestRoleHandler_Get(t *testing.T) {
	r, token := newAppServer(t)
	// 列表拿首个 id
	wl := authedReq(t, r, http.MethodGet, "/api/role?page=1&size=10", token, nil)
	require.Equal(t, 200, wl.Code, wl.Body.String())
	ld, _ := decodeResult(t, wl).Data.(map[string]any)
	records, _ := ld["records"].([]any)
	require.NotEmpty(t, records, "seed 后角色列表非空")
	first, _ := records[0].(map[string]any)
	id := strconv.Itoa(int(first["id"].(float64)))

	w := authedReq(t, r, http.MethodGet, "/api/role/"+id, token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	d, _ := decodeResult(t, w).Data.(map[string]any)
	assert.NotEmpty(t, d["code"], "详情应返回 code 字段")
}

// TestRoleHandler_CRUD 创建→查询→更新→删除 主链路。
func TestRoleHandler_CRUD(t *testing.T) {
	r, token := newAppServer(t)

	// 创建
	body := roleCreateReq{
		Name:   "测试角色",
		Code:   "test:role:crud",
		Status: "active",
	}
	w := authedReq(t, r, http.MethodPost, "/api/role", token, body)
	require.Equalf(t, 200, w.Code, "create body=%s", w.Body.String())
	d, _ := decodeResult(t, w).Data.(map[string]any)
	id := strconv.Itoa(int(d["id"].(float64)))

	// 查询
	w = authedReq(t, r, http.MethodGet, "/api/role/"+id, token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	d, _ = decodeResult(t, w).Data.(map[string]any)
	assert.Equal(t, "test:role:crud", d["code"])

	// 更新
	body.Name = "测试角色改名"
	w = authedReq(t, r, http.MethodPut, "/api/role/"+id, token, body)
	require.Equal(t, 200, w.Code, w.Body.String())

	// 删除
	w = authedReq(t, r, http.MethodDelete, "/api/role/"+id, token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())

	// 删后 get → 404
	w = authedReq(t, r, http.MethodGet, "/api/role/"+id, token, nil)
	assert.Equal(t, 404, w.Code)
}

// TestRoleHandler_BatchDelete 先创建 2 个角色再批量删除。
func TestRoleHandler_BatchDelete(t *testing.T) {
	r, token := newAppServer(t)
	ids := make([]string, 0, 2)
	for _, code := range []string{"test:role:batch:1", "test:role:batch:2"} {
		w := authedReq(t, r, http.MethodPost, "/api/role", token, roleCreateReq{
			Name:   code,
			Code:   code,
			Status: "active",
		})
		require.Equal(t, 200, w.Code, w.Body.String())
		d, _ := decodeResult(t, w).Data.(map[string]any)
		ids = append(ids, strconv.Itoa(int(d["id"].(float64))))
	}
	w := authedReq(t, r, http.MethodDelete, "/api/role", token, batchDeleteReq{IDs: ids})
	require.Equal(t, 200, w.Code, w.Body.String())
}

// TestRoleHandler_Export 导出 CSV，msg="导出成功"。
func TestRoleHandler_Export(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/role/export", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Equal(t, "导出成功", decodeResult(t, w).Msg)
}

// TestRoleHandler_GetPermissions 查询角色已分配权限 code 数组。
func TestRoleHandler_GetPermissions(t *testing.T) {
	r, token := newAppServer(t)
	// 列表拿首个 id
	wl := authedReq(t, r, http.MethodGet, "/api/role?page=1&size=10", token, nil)
	require.Equal(t, 200, wl.Code, wl.Body.String())
	ld, _ := decodeResult(t, wl).Data.(map[string]any)
	records, _ := ld["records"].([]any)
	require.NotEmpty(t, records)
	first, _ := records[0].(map[string]any)
	id := strconv.Itoa(int(first["id"].(float64)))

	w := authedReq(t, r, http.MethodGet, "/api/role/"+id+"/permissions", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	codes, ok := decodeResult(t, w).Data.([]any)
	assert.True(t, ok, "GetPermissions 应返回数组")
	_ = codes
}

// TestRoleHandler_SetPermissions 分配角色权限：先取一个真实权限 code，再设置并回查。
// service.SetPermissions 严格校验 code 必须存在，故需先从权限列表取真实 code。
func TestRoleHandler_SetPermissions(t *testing.T) {
	r, token := newAppServer(t)

	// 取一个真实存在的权限 code
	wp := authedReq(t, r, http.MethodGet, "/api/permission?all=true", token, nil)
	require.Equal(t, 200, wp.Code, wp.Body.String())
	perms, _ := decodeResult(t, wp).Data.([]any)
	require.NotEmpty(t, perms, "seed 后权限列表非空")
	firstPerm, _ := perms[0].(map[string]any)
	code, _ := firstPerm["code"].(string)
	require.NotEmpty(t, code, "权限 code 非空")

	// 创建一个角色用于权限分配
	wc := authedReq(t, r, http.MethodPost, "/api/role", token, roleCreateReq{
		Name:   "权限测试角色",
		Code:   "test:role:perm",
		Status: "active",
	})
	require.Equal(t, 200, wc.Code, wc.Body.String())
	d, _ := decodeResult(t, wc).Data.(map[string]any)
	id := strconv.Itoa(int(d["id"].(float64)))

	// 设置权限
	ws := authedReq(t, r, http.MethodPut, "/api/role/"+id+"/permissions", token,
		setPermissionsReq{Permissions: []string{code}})
	require.Equal(t, 200, ws.Code, ws.Body.String())
	assert.Equal(t, "设置成功", decodeResult(t, ws).Msg)

	// 回查应包含已分配 code
	wg := authedReq(t, r, http.MethodGet, "/api/role/"+id+"/permissions", token, nil)
	require.Equal(t, 200, wg.Code, wg.Body.String())
	codes, _ := decodeResult(t, wg).Data.([]any)
	assert.Contains(t, codes, code)
}

// TestRoleHandler_NoToken 未鉴权 → 401。
func TestRoleHandler_NoToken(t *testing.T) {
	r, _ := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/role", "", nil)
	assert.Equal(t, 401, w.Code)
}

// TestRoleHandler_Validation 缺必填字段 → 422。
func TestRoleHandler_Validation(t *testing.T) {
	r, token := newAppServer(t)
	// 缺 code/status
	w := authedReq(t, r, http.MethodPost, "/api/role", token, roleCreateReq{Name: "x"})
	assert.Equal(t, 422, w.Code)
}
