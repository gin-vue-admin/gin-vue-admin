package handler

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDashboardHandler_Stats 首页统计卡片（seed 后 data 非空）。
func TestDashboardHandler_Stats(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/dashboard/stats", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.NotNil(t, decodeResult(t, w).Data, "stats data 非空")
}

// TestDashboardHandler_Charts 首页图表（趋势 + 分布）。
func TestDashboardHandler_Charts(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/dashboard/charts?range=7d", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
}

// TestDashboardHandler_Activities 最近活动列表。
func TestDashboardHandler_Activities(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/dashboard/activities", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
}

// TestHealthHandler_Health 健康检查（无需鉴权，用 doJSON 不带 token）。
func TestHealthHandler_Health(t *testing.T) {
	r, _ := newAppServer(t)
	w := doJSON(t, r, http.MethodGet, "/api/health", nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Equal(t, "up", decodeResult(t, w).Data.(map[string]any)["status"])
}

// TestCrudHandler_CRUD 创建→查询→更新→删除 主链路（删后 get → 404）。
func TestCrudHandler_CRUD(t *testing.T) {
	r, token := newAppServer(t)

	// 创建
	body := crudUpsertReq{
		Date: "2026-07-07", Name: "测试条目", Province: "广东",
		City: "深圳", Address: "南山", Zip: 518000,
	}
	w := authedReq(t, r, http.MethodPost, "/api/crud", token, body)
	require.Equalf(t, 200, w.Code, "create body=%s", w.Body.String())
	d, _ := decodeResult(t, w).Data.(map[string]any)
	id := strconv.Itoa(int(d["id"].(float64)))

	// 查询
	w = authedReq(t, r, http.MethodGet, "/api/crud/"+id, token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	d, _ = decodeResult(t, w).Data.(map[string]any)
	assert.Equal(t, "测试条目", d["name"])

	// 更新
	body.Name = "测试条目改名"
	body.City = "广州"
	w = authedReq(t, r, http.MethodPut, "/api/crud/"+id, token, body)
	require.Equal(t, 200, w.Code, w.Body.String())

	// 删除
	w = authedReq(t, r, http.MethodDelete, "/api/crud/"+id, token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())

	// 删后 get → 404
	w = authedReq(t, r, http.MethodGet, "/api/crud/"+id, token, nil)
	assert.Equal(t, 404, w.Code)
}

// TestCrudHandler_BatchDelete 先创建 2 条再批量删除。
func TestCrudHandler_BatchDelete(t *testing.T) {
	r, token := newAppServer(t)
	ids := make([]string, 0, 2)
	for _, name := range []string{"批量1", "批量2"} {
		w := authedReq(t, r, http.MethodPost, "/api/crud", token, crudUpsertReq{
			Date: "2026-07-07", Name: name, Province: "广东", City: "深圳", Zip: 518000,
		})
		require.Equal(t, 200, w.Code, w.Body.String())
		d, _ := decodeResult(t, w).Data.(map[string]any)
		ids = append(ids, strconv.Itoa(int(d["id"].(float64))))
	}
	w := authedReq(t, r, http.MethodDelete, "/api/crud", token, batchDeleteReq{IDs: ids})
	require.Equal(t, 200, w.Code, w.Body.String())
}

// TestCrudHandler_NoToken 未鉴权访问 crud 列表 → 401。
func TestCrudHandler_NoToken(t *testing.T) {
	r, _ := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/crud", "", nil)
	assert.Equal(t, 401, w.Code)
}

// TestDashboardHandler_NoToken 未鉴权访问 dashboard stats → 401。
func TestDashboardHandler_NoToken(t *testing.T) {
	r, _ := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/dashboard/stats", "", nil)
	assert.Equal(t, 401, w.Code)
}
