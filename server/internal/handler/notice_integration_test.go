package handler

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noticeCreateBody 构造公告创建/更新请求体（字段对齐 service.NoticeUpsertReq）。
// 用 map[string]any 避免 handler 测试反向依赖 service 包。
func noticeCreateBody(title string) map[string]any {
	return map[string]any{
		"title":   title,
		"content": "内容",
		"type":    "notice",
		"status":  "draft",
	}
}

// TestNoticeHandler_CRUD 创建→查询→更新→删除 主链路。
func TestNoticeHandler_CRUD(t *testing.T) {
	r, token := newAppServer(t)

	// 创建
	body := noticeCreateBody("测试公告")
	w := authedReq(t, r, http.MethodPost, "/api/system/notice", token, body)
	require.Equalf(t, 200, w.Code, "create body=%s", w.Body.String())
	d, _ := decodeResult(t, w).Data.(map[string]any)
	id := strconv.Itoa(int(d["id"].(float64)))

	// 查询
	w = authedReq(t, r, http.MethodGet, "/api/system/notice/"+id, token, nil)
	require.Equal(t, 200, w.Code)
	d, _ = decodeResult(t, w).Data.(map[string]any)
	assert.Equal(t, "测试公告", d["title"])

	// 更新
	body["title"] = "测试公告改名"
	w = authedReq(t, r, http.MethodPut, "/api/system/notice/"+id, token, body)
	require.Equal(t, 200, w.Code, w.Body.String())

	// 删除
	w = authedReq(t, r, http.MethodDelete, "/api/system/notice/"+id, token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())

	// 删后 get → 404
	w = authedReq(t, r, http.MethodGet, "/api/system/notice/"+id, token, nil)
	assert.Equal(t, 404, w.Code)
}

// TestNoticeHandler_Publish 发布公告：status → published，并记录发布时间。
func TestNoticeHandler_Publish(t *testing.T) {
	r, token := newAppServer(t)

	// 创建（draft）
	w := authedReq(t, r, http.MethodPost, "/api/system/notice", token, noticeCreateBody("发布公告"))
	require.Equalf(t, 200, w.Code, "create body=%s", w.Body.String())
	d, _ := decodeResult(t, w).Data.(map[string]any)
	id := strconv.Itoa(int(d["id"].(float64)))

	// 发布
	w = authedReq(t, r, http.MethodPost, "/api/system/notice/"+id+"/publish", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Equal(t, "发布成功", decodeResult(t, w).Msg)

	// 校验状态变更 + 发布时间已写
	w = authedReq(t, r, http.MethodGet, "/api/system/notice/"+id, token, nil)
	require.Equal(t, 200, w.Code)
	d, _ = decodeResult(t, w).Data.(map[string]any)
	assert.Equal(t, "published", d["status"])
	assert.NotEmpty(t, d["publishTime"], "发布后 publishTime 非空")
}

// TestNoticeHandler_Revoke 撤销公告：status → draft。
func TestNoticeHandler_Revoke(t *testing.T) {
	r, token := newAppServer(t)

	// 创建（published）
	body := noticeCreateBody("撤销公告")
	body["status"] = "published"
	w := authedReq(t, r, http.MethodPost, "/api/system/notice", token, body)
	require.Equalf(t, 200, w.Code, "create body=%s", w.Body.String())
	d, _ := decodeResult(t, w).Data.(map[string]any)
	id := strconv.Itoa(int(d["id"].(float64)))

	// 撤销
	w = authedReq(t, r, http.MethodPost, "/api/system/notice/"+id+"/revoke", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Equal(t, "撤销成功", decodeResult(t, w).Msg)

	// 校验状态变更
	w = authedReq(t, r, http.MethodGet, "/api/system/notice/"+id, token, nil)
	require.Equal(t, 200, w.Code)
	d, _ = decodeResult(t, w).Data.(map[string]any)
	assert.Equal(t, "draft", d["status"])
}

// TestNoticeHandler_List 分页查询公告列表（先创建一条确保非空）。
func TestNoticeHandler_List(t *testing.T) {
	r, token := newAppServer(t)

	// 先创建一条确保列表非空
	w := authedReq(t, r, http.MethodPost, "/api/system/notice", token, noticeCreateBody("列表公告"))
	require.Equalf(t, 200, w.Code, "create body=%s", w.Body.String())

	// 列表
	w = authedReq(t, r, http.MethodGet, "/api/system/notice?page=1&size=10", token, nil)
	require.Equal(t, 200, w.Code)
	data, _ := decodeResult(t, w).Data.(map[string]any)
	records, _ := data["records"].([]any)
	assert.NotEmpty(t, records, "创建后列表非空")
}

// TestNoticeHandler_Export 导出 CSV。
func TestNoticeHandler_Export(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/system/notice/export", token, nil)
	require.Equal(t, 200, w.Code)
	assert.Equal(t, "导出成功", decodeResult(t, w).Msg)
}

// TestNoticeHandler_BatchDelete 批量删除。
func TestNoticeHandler_BatchDelete(t *testing.T) {
	r, token := newAppServer(t)
	ids := make([]string, 0, 2)
	for _, title := range []string{"批量公告1", "批量公告2"} {
		w := authedReq(t, r, http.MethodPost, "/api/system/notice", token, noticeCreateBody(title))
		require.Equal(t, 200, w.Code, w.Body.String())
		d, _ := decodeResult(t, w).Data.(map[string]any)
		ids = append(ids, strconv.Itoa(int(d["id"].(float64))))
	}
	w := authedReq(t, r, http.MethodDelete, "/api/system/notice", token, batchDeleteReq{IDs: ids})
	require.Equal(t, 200, w.Code, w.Body.String())
}

// TestNoticeHandler_NoToken 未鉴权 → 401。
func TestNoticeHandler_NoToken(t *testing.T) {
	r, _ := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/system/notice", "", nil)
	assert.Equal(t, 401, w.Code)
}

// TestNoticeHandler_Validation 缺必填字段（title）→ 422。
func TestNoticeHandler_Validation(t *testing.T) {
	r, token := newAppServer(t)
	// 缺 title（binding:"required"）
	w := authedReq(t, r, http.MethodPost, "/api/system/notice", token, map[string]any{
		"content": "无标题",
	})
	assert.Equal(t, 422, w.Code)
}
