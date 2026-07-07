package handler

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedDictCategory 通过 admin 创建一个字典分类，返回其 id（字符串）。
// 供二、三级测试建立上级依赖复用。
func seedDictCategory(t *testing.T, r http.Handler, token, code string) string {
	t.Helper()
	w := authedReq(t, r, http.MethodPost, "/api/dict/categories", token, map[string]any{
		"name": "分类-" + code, "code": code, "description": "", "status": "active",
	})
	require.Equalf(t, 200, w.Code, "create category body=%s", w.Body.String())
	d, _ := decodeResult(t, w).Data.(map[string]any)
	return strconv.Itoa(int(d["id"].(float64)))
}

// seedDict 通过 admin 在指定分类下创建一个字典，返回其 id（字符串）。
// 供三级（字典项）测试建立上级依赖复用。
func seedDict(t *testing.T, r http.Handler, token, categoryID, code string) string {
	t.Helper()
	catID, err := strconv.Atoi(categoryID) // categoryId 为 uint，body 需传数字
	require.NoError(t, err)
	w := authedReq(t, r, http.MethodPost, "/api/dict/dicts", token, map[string]any{
		"categoryId": catID, "name": "字典-" + code, "code": code, "description": "", "status": "active",
	})
	require.Equalf(t, 200, w.Code, "create dict body=%s", w.Body.String())
	d, _ := decodeResult(t, w).Data.(map[string]any)
	return strconv.Itoa(int(d["id"].(float64)))
}

// ==================== Level 1: 字典分类 ====================

// TestDictCategoryHandler_CRUD 分类 Create→Get→List→Update→Delete 主链路。
func TestDictCategoryHandler_CRUD(t *testing.T) {
	r, token := newAppServer(t)

	// 创建
	w := authedReq(t, r, http.MethodPost, "/api/dict/categories", token, map[string]any{
		"name": "性别", "code": "gender", "description": "性别字典分类", "status": "active",
	})
	require.Equalf(t, 200, w.Code, "create body=%s", w.Body.String())
	d, _ := decodeResult(t, w).Data.(map[string]any)
	id := strconv.Itoa(int(d["id"].(float64)))
	assert.Equal(t, "gender", d["code"])

	// 查询
	w = authedReq(t, r, http.MethodGet, "/api/dict/categories/"+id, token, nil)
	require.Equal(t, 200, w.Code)
	d, _ = decodeResult(t, w).Data.(map[string]any)
	assert.Equal(t, "性别", d["name"])

	// 列表
	w = authedReq(t, r, http.MethodGet, "/api/dict/categories?page=1&size=10", token, nil)
	require.Equal(t, 200, w.Code)
	list, _ := decodeResult(t, w).Data.(map[string]any)
	records, _ := list["records"].([]any)
	assert.NotEmpty(t, records, "创建后分类列表非空")

	// 更新
	w = authedReq(t, r, http.MethodPut, "/api/dict/categories/"+id, token, map[string]any{
		"name": "性别分类", "code": "gender", "description": "改名后", "status": "inactive",
	})
	require.Equal(t, 200, w.Code, w.Body.String())
	w = authedReq(t, r, http.MethodGet, "/api/dict/categories/"+id, token, nil)
	require.Equal(t, 200, w.Code)
	d, _ = decodeResult(t, w).Data.(map[string]any)
	assert.Equal(t, "性别分类", d["name"])
	assert.Equal(t, "inactive", d["status"])

	// 删除
	w = authedReq(t, r, http.MethodDelete, "/api/dict/categories/"+id, token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())

	// 删后 get → 404
	w = authedReq(t, r, http.MethodGet, "/api/dict/categories/"+id, token, nil)
	assert.Equal(t, 404, w.Code)
}

// TestDictCategoryHandler_BatchDelete 批量删除分类。
func TestDictCategoryHandler_BatchDelete(t *testing.T) {
	r, token := newAppServer(t)
	ids := []string{
		seedDictCategory(t, r, token, "batch-c1"),
		seedDictCategory(t, r, token, "batch-c2"),
	}
	w := authedReq(t, r, http.MethodDelete, "/api/dict/categories", token, batchDeleteReq{IDs: ids})
	require.Equal(t, 200, w.Code, w.Body.String())
}

// TestDictCategoryHandler_Export 导出 CSV。
func TestDictCategoryHandler_Export(t *testing.T) {
	r, token := newAppServer(t)
	seedDictCategory(t, r, token, "export-c")
	w := authedReq(t, r, http.MethodGet, "/api/dict/categories/export", token, nil)
	require.Equal(t, 200, w.Code)
	assert.Equal(t, "导出成功", decodeResult(t, w).Msg)
}

// TestDictCategoryHandler_NoToken 未鉴权访问 → 401。
func TestDictCategoryHandler_NoToken(t *testing.T) {
	r, _ := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/dict/categories", "", nil)
	assert.Equal(t, 401, w.Code)
}

// ==================== Level 2: 字典 ====================

// TestDictHandler_CreateAndList 二级字典 创建 + 列表（验证二级能跑通）。
func TestDictHandler_CreateAndList(t *testing.T) {
	r, token := newAppServer(t)
	categoryID := seedDictCategory(t, r, token, "dict-cl")
	catID, err := strconv.Atoi(categoryID) // categoryId 为 uint，body 需传数字
	require.NoError(t, err)

	// 创建字典
	w := authedReq(t, r, http.MethodPost, "/api/dict/dicts", token, map[string]any{
		"categoryId": catID, "name": "用户性别", "code": "user_gender",
		"description": "", "status": "active",
	})
	require.Equalf(t, 200, w.Code, "create dict body=%s", w.Body.String())
	d, _ := decodeResult(t, w).Data.(map[string]any)
	assert.Equal(t, "user_gender", d["code"])

	// 列表
	w = authedReq(t, r, http.MethodGet, "/api/dict/dicts?page=1&size=10", token, nil)
	require.Equal(t, 200, w.Code)
	list, _ := decodeResult(t, w).Data.(map[string]any)
	records, _ := list["records"].([]any)
	assert.NotEmpty(t, records, "字典列表非空")

	// 按 categoryId 过滤应命中
	w = authedReq(t, r, http.MethodGet, "/api/dict/dicts?page=1&size=10&categoryId="+categoryID, token, nil)
	require.Equal(t, 200, w.Code)
	list, _ = decodeResult(t, w).Data.(map[string]any)
	records, _ = list["records"].([]any)
	assert.NotEmpty(t, records, "按分类过滤应命中")
}

// ==================== Level 3: 字典项 ====================

// TestDictItemHandler_CreateAndList 三级字典项 创建 + 列表（验证三级能跑通）。
// items 需要先有 dictId，故先建分类→字典两级上级。
func TestDictItemHandler_CreateAndList(t *testing.T) {
	r, token := newAppServer(t)
	categoryID := seedDictCategory(t, r, token, "item-cl")
	dictID := seedDict(t, r, token, categoryID, "item-dl")
	dictIDI, err := strconv.Atoi(dictID) // dictId 为 uint，body 需传数字
	require.NoError(t, err)

	// 创建字典项
	w := authedReq(t, r, http.MethodPost, "/api/dict/items", token, map[string]any{
		"dictId": dictIDI, "name": "男", "code": "male", "value": "1", "sort": 1, "status": "active",
	})
	require.Equalf(t, 200, w.Code, "create item body=%s", w.Body.String())
	d, _ := decodeResult(t, w).Data.(map[string]any)
	assert.Equal(t, "male", d["code"])

	// 列表
	w = authedReq(t, r, http.MethodGet, "/api/dict/items?page=1&size=10", token, nil)
	require.Equal(t, 200, w.Code)
	list, _ := decodeResult(t, w).Data.(map[string]any)
	records, _ := list["records"].([]any)
	assert.NotEmpty(t, records, "字典项列表非空")

	// 按 dictId 过滤应命中
	w = authedReq(t, r, http.MethodGet, "/api/dict/items?page=1&size=10&dictId="+dictID, token, nil)
	require.Equal(t, 200, w.Code)
	list, _ = decodeResult(t, w).Data.(map[string]any)
	records, _ = list["records"].([]any)
	assert.NotEmpty(t, records, "按字典过滤应命中")
}
