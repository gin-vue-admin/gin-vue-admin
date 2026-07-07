package handler

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeptHandler_List 先建一个根部门，再查询树形列表（返回数组且非空）。
// 注：DeptService 无 seed，需自行造数据。
func TestDeptHandler_List(t *testing.T) {
	r, token := newAppServer(t)

	// 先建一个根部门，保证列表非空
	w := authedReq(t, r, http.MethodPost, "/api/system/dept", token, map[string]any{
		"name": "列表测试部", "status": "active", "sort": 0,
	})
	require.Equalf(t, 200, w.Code, "create body=%s", w.Body.String())

	// List 返回树形数组（非分页）
	w = authedReq(t, r, http.MethodGet, "/api/system/dept", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	list, _ := decodeResult(t, w).Data.([]any)
	assert.NotEmpty(t, list, "创建后部门树非空")
}

// TestDeptHandler_CRUD 创建→查询→更新→删除 主链路（删后 get 404）。
func TestDeptHandler_CRUD(t *testing.T) {
	r, token := newAppServer(t)

	// 创建
	w := authedReq(t, r, http.MethodPost, "/api/system/dept", token, map[string]any{
		"name": "主链路部", "leader": "张三", "phone": "13800000000",
		"email": "z@a.com", "sort": 1, "status": "active",
	})
	require.Equalf(t, 200, w.Code, "create body=%s", w.Body.String())
	d, _ := decodeResult(t, w).Data.(map[string]any)
	id := strconv.Itoa(int(d["id"].(float64)))

	// 查询
	w = authedReq(t, r, http.MethodGet, "/api/system/dept/"+id, token, nil)
	require.Equal(t, 200, w.Code)
	d, _ = decodeResult(t, w).Data.(map[string]any)
	assert.Equal(t, "主链路部", d["name"])

	// 更新
	w = authedReq(t, r, http.MethodPut, "/api/system/dept/"+id, token, map[string]any{
		"name": "主链路部改名", "leader": "李四", "sort": 2, "status": "active",
	})
	require.Equal(t, 200, w.Code, w.Body.String())

	// 更新后 get 验证字段已落库
	w = authedReq(t, r, http.MethodGet, "/api/system/dept/"+id, token, nil)
	require.Equal(t, 200, w.Code)
	d, _ = decodeResult(t, w).Data.(map[string]any)
	assert.Equal(t, "主链路部改名", d["name"])
	assert.Equal(t, "李四", d["leader"])

	// 删除（无子节点，单点删）
	w = authedReq(t, r, http.MethodDelete, "/api/system/dept/"+id, token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())

	// 删后 get → 404
	w = authedReq(t, r, http.MethodGet, "/api/system/dept/"+id, token, nil)
	assert.Equal(t, 404, w.Code)
}

// TestDeptHandler_BatchDelete 批量删除（先建 2 个根部门再批量删）。
func TestDeptHandler_BatchDelete(t *testing.T) {
	r, token := newAppServer(t)
	ids := make([]string, 0, 2)
	for _, name := range []string{"批量部A", "批量部B"} {
		w := authedReq(t, r, http.MethodPost, "/api/system/dept", token, map[string]any{
			"name": name, "status": "active",
		})
		require.Equal(t, 200, w.Code, w.Body.String())
		d, _ := decodeResult(t, w).Data.(map[string]any)
		ids = append(ids, strconv.Itoa(int(d["id"].(float64))))
	}
	w := authedReq(t, r, http.MethodDelete, "/api/system/dept", token, batchDeleteReq{IDs: ids})
	require.Equal(t, 200, w.Code, w.Body.String())
}

// TestDeptHandler_Export 导出 CSV（msg="导出成功"）。
func TestDeptHandler_Export(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/system/dept/export", token, nil)
	require.Equal(t, 200, w.Code)
	assert.Equal(t, "导出成功", decodeResult(t, w).Msg)
}

// TestDeptHandler_NoToken 未鉴权 → 401。
func TestDeptHandler_NoToken(t *testing.T) {
	r, _ := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/system/dept", "", nil)
	assert.Equal(t, 401, w.Code)
}

// TestDeptHandler_Validation 缺必填字段（name）→ 422。
func TestDeptHandler_Validation(t *testing.T) {
	r, token := newAppServer(t)
	// 缺 name（binding:"required"）
	w := authedReq(t, r, http.MethodPost, "/api/system/dept", token, map[string]any{
		"status": "active",
	})
	assert.Equal(t, 422, w.Code)
}
