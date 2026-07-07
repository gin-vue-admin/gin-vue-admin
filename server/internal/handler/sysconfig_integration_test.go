package handler

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSysConfigHandler_List admin 分页查询系统配置列表（seed 后非空）。
func TestSysConfigHandler_List(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/system/config?page=1&size=10", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	data, _ := decodeResult(t, w).Data.(map[string]any)
	records, _ := data["records"].([]any)
	assert.NotEmpty(t, records, "seed 后系统配置列表非空")
}

// TestSysConfigHandler_GetByKey 按 key 取配置（路由 /key/:key 在 /:id 前，site_title 由 seed 注入）。
func TestSysConfigHandler_GetByKey(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/system/config/key/site_title", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	d, _ := decodeResult(t, w).Data.(map[string]any)
	assert.Equal(t, "site_title", d["configKey"])
	assert.NotEmpty(t, d["configValue"], "seed 的 site_title 应有值")
}

// TestSysConfigHandler_CRUD 创建→查询→更新→删除 主链路（configKey 用 test:cfg:xxx 避免与 seed 冲突）。
func TestSysConfigHandler_CRUD(t *testing.T) {
	r, token := newAppServer(t)

	// 创建
	body := sysConfigUpsertReq{
		ConfigKey: "test:cfg:integration", ConfigValue: "v1",
		ConfigName: "测试配置", Type: "string",
	}
	w := authedReq(t, r, http.MethodPost, "/api/system/config", token, body)
	require.Equalf(t, 200, w.Code, "create body=%s", w.Body.String())
	d, _ := decodeResult(t, w).Data.(map[string]any)
	id := strconv.Itoa(int(d["id"].(float64)))

	// 按 id 查询
	w = authedReq(t, r, http.MethodGet, "/api/system/config/"+id, token, nil)
	require.Equal(t, 200, w.Code)
	d, _ = decodeResult(t, w).Data.(map[string]any)
	assert.Equal(t, "test:cfg:integration", d["configKey"])

	// 更新
	body.ConfigValue = "v2"
	body.ConfigName = "测试配置改名"
	w = authedReq(t, r, http.MethodPut, "/api/system/config/"+id, token, body)
	require.Equal(t, 200, w.Code, w.Body.String())

	// 删除
	w = authedReq(t, r, http.MethodDelete, "/api/system/config/"+id, token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())

	// 删后 get → 404
	w = authedReq(t, r, http.MethodGet, "/api/system/config/"+id, token, nil)
	assert.Equal(t, 404, w.Code)
}

// TestSysConfigHandler_GetPublic 公开配置无需鉴权（前端启动拉取），用 doJSON 不带 token 应 200。
func TestSysConfigHandler_GetPublic(t *testing.T) {
	r, _ := newAppServer(t)
	w := doJSON(t, r, http.MethodGet, "/api/system/config/public", nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	d, _ := decodeResult(t, w).Data.(map[string]any)
	assert.Contains(t, d, "site_title", "公开配置应包含 site_title")
}

// TestSysConfigHandler_NoToken 未鉴权访问受保护端点 → 401。
func TestSysConfigHandler_NoToken(t *testing.T) {
	r, _ := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/system/config", "", nil)
	assert.Equal(t, 401, w.Code)
}

// TestSysConfigHandler_Validation 缺必填字段 configKey → 422。
func TestSysConfigHandler_Validation(t *testing.T) {
	r, token := newAppServer(t)
	// 缺 configKey（required），configValue 亦缺
	w := authedReq(t, r, http.MethodPost, "/api/system/config", token, sysConfigUpsertReq{})
	assert.Equal(t, 422, w.Code)
}
