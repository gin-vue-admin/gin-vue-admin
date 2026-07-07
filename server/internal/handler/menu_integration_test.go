package handler

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findMenuNode 在菜单树（[]any，元素为 map[string]any）中按 id 递归查找节点。
// GetTree 返回的 MenuInfo 树用 JSON 反序列化后即此结构。
func findMenuNode(tree []any, id float64) map[string]any {
	for _, node := range tree {
		n, ok := node.(map[string]any)
		if !ok {
			continue
		}
		if nid, ok := n["id"].(float64); ok && nid == id {
			return n
		}
		if children, ok := n["children"].([]any); ok {
			if found := findMenuNode(children, id); found != nil {
				return found
			}
		}
	}
	return nil
}

// TestMenuHandler_Menus GET /api/system/menus 返回当前用户菜单树（MenuDTO，seed 后非空）。
func TestMenuHandler_Menus(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/system/menus", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	tree, _ := decodeResult(t, w).Data.([]any)
	assert.NotEmpty(t, tree, "seed 后菜单树非空")
}

// TestMenuHandler_GetTree GET /api/system/menu 返回管理用菜单树（MenuInfo，含 id/parentId/sort/status）。
func TestMenuHandler_GetTree(t *testing.T) {
	r, token := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/system/menu", token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	tree, _ := decodeResult(t, w).Data.([]any)
	assert.NotEmpty(t, tree, "seed 后管理菜单树非空")
	first, _ := tree[0].(map[string]any)
	require.NotNil(t, first, "首节点应为对象")
	assert.NotNil(t, first["id"], "MenuInfo 应含 id 字段")
}

// TestMenuHandler_CRUD 创建→GetTree 可见→更新→删除 主链路。
func TestMenuHandler_CRUD(t *testing.T) {
	r, token := newAppServer(t)

	// 创建
	createBody := map[string]any{
		"name":      "test-menu",
		"title":     "测试菜单",
		"path":      "/test",
		"component": "test/views/List",
		"icon":      "Document",
		"sort":      0,
		"status":    "active",
	}
	w := authedReq(t, r, http.MethodPost, "/api/system/menu", token, createBody)
	require.Equalf(t, 200, w.Code, "create body=%s", w.Body.String())
	d, _ := decodeResult(t, w).Data.(map[string]any)
	id := strconv.Itoa(int(d["id"].(float64)))
	idFloat := d["id"].(float64)

	// GetTree 可见
	w = authedReq(t, r, http.MethodGet, "/api/system/menu", token, nil)
	require.Equal(t, 200, w.Code)
	tree, _ := decodeResult(t, w).Data.([]any)
	assert.NotNil(t, findMenuNode(tree, idFloat), "新建菜单应在树中可见")

	// 更新（title 改名）
	updateBody := map[string]any{
		"name":      "test-menu",
		"title":     "测试菜单改名",
		"path":      "/test",
		"component": "test/views/List",
		"icon":      "Document",
		"sort":      0,
		"status":    "active",
	}
	w = authedReq(t, r, http.MethodPut, "/api/system/menu/"+id, token, updateBody)
	require.Equal(t, 200, w.Code, w.Body.String())

	// 验证更新生效
	w = authedReq(t, r, http.MethodGet, "/api/system/menu", token, nil)
	require.Equal(t, 200, w.Code)
	tree, _ = decodeResult(t, w).Data.([]any)
	node := findMenuNode(tree, idFloat)
	require.NotNil(t, node, "更新后菜单仍应在树中")
	assert.Equal(t, "测试菜单改名", node["title"], "title 应已更新")

	// 删除
	w = authedReq(t, r, http.MethodDelete, "/api/system/menu/"+id, token, nil)
	require.Equal(t, 200, w.Code, w.Body.String())

	// 删后 GetTree 不可见
	w = authedReq(t, r, http.MethodGet, "/api/system/menu", token, nil)
	require.Equal(t, 200, w.Code)
	tree, _ = decodeResult(t, w).Data.([]any)
	assert.Nil(t, findMenuNode(tree, idFloat), "删除后菜单不应在树中")
}

// TestMenuHandler_Sort 拖拽排序（inner：dragging 变 target 的子节点）。
func TestMenuHandler_Sort(t *testing.T) {
	r, token := newAppServer(t)

	// 创建两个根菜单作为拖拽/目标
	mk := func(name, path string) float64 {
		w := authedReq(t, r, http.MethodPost, "/api/system/menu", token, map[string]any{
			"name": name, "title": name, "path": path, "status": "active",
		})
		require.Equal(t, 200, w.Code, w.Body.String())
		d, _ := decodeResult(t, w).Data.(map[string]any)
		return d["id"].(float64)
	}
	id1 := mk("sort-target", "/sort-target")
	id2 := mk("sort-drag", "/sort-drag")

	// Sort inner：id2 拖入 id1 内部
	w := authedReq(t, r, http.MethodPatch, "/api/system/menu/sort", token, map[string]any{
		"draggingId": id2, "targetId": id1, "position": "inner",
	})
	require.Equal(t, 200, w.Code, w.Body.String())
	assert.Equal(t, "排序成功", decodeResult(t, w).Msg)

	// GetTree 验证：id2 的 parentId 应变为 id1
	w = authedReq(t, r, http.MethodGet, "/api/system/menu", token, nil)
	require.Equal(t, 200, w.Code)
	tree, _ := decodeResult(t, w).Data.([]any)
	node := findMenuNode(tree, id2)
	require.NotNil(t, node, "拖拽节点应在树中")
	parentID, _ := node["parentId"].(float64)
	assert.Equal(t, id1, parentID, "inner 后 dragging.parentId 应等于 target.id")
}

// TestMenuHandler_NoToken 未鉴权访问 → 401。
func TestMenuHandler_NoToken(t *testing.T) {
	r, _ := newAppServer(t)
	w := authedReq(t, r, http.MethodGet, "/api/system/menu", "", nil)
	assert.Equal(t, 401, w.Code)
}
