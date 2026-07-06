package handler

import (
	"strconv"

	"gva/internal/pkg/apperr"
	"gva/internal/pkg/response"
	"gva/internal/service"

	"github.com/gin-gonic/gin"
)

// MenuHandler 菜单端点。
type MenuHandler struct {
	svc *service.MenuService
}

func NewMenuHandler(svc *service.MenuService) *MenuHandler {
	return &MenuHandler{svc: svc}
}

// Menus GET /api/system/menus —— 当前用户菜单树（后端下发完整树，前端按权限过滤）。
// @Summary      当前用户菜单树
// @Tags         menu
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} response.ApiResult
// @Router       /system/menus [get]
func (h *MenuHandler) Menus(c *gin.Context) {
	tree, err := h.svc.GetMenus(c.Request.Context())
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, tree)
}

// GetTree GET /api/system/menu —— 管理用菜单树（MenuInfo，含 id/parentId/sort/status）。
// @Summary      管理用菜单树
// @Tags         menu
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} response.ApiResult
// @Router       /system/menu [get]
func (h *MenuHandler) GetTree(c *gin.Context) {
	tree, err := h.svc.GetTree(c.Request.Context())
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, tree)
}

// Create POST /api/system/menu —— 创建菜单。
// @Summary      创建菜单
// @Tags         menu
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body service.MenuCreateReq true "菜单信息"
// @Success      200  {object} response.ApiResult
// @Router       /system/menu [post]
func (h *MenuHandler) Create(c *gin.Context) {
	var req service.MenuCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	m, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, m, "创建成功")
}

// Update PUT /api/system/menu/:id —— 更新菜单（先 Get 再改字段再 Update）。
// @Summary      更新菜单
// @Tags         menu
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path int true "菜单 ID"
// @Param        request body service.MenuCreateReq true "菜单信息"
// @Success      200  {object} response.ApiResult
// @Failure      404  {object} response.ProblemDetail
// @Router       /system/menu/{id} [put]
func (h *MenuHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	m, err := h.svc.Get(c.Request.Context(), uint(id))
	if err != nil {
		apperr.Write(c, err)
		return
	}
	var req service.MenuCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	m.Name = req.Name
	m.Title = req.Title
	m.Path = req.Path
	m.Component = req.Component
	m.Icon = req.Icon
	m.Sort = req.Sort
	m.Status = req.Status
	if req.ParentID != nil {
		m.ParentID = *req.ParentID
	} else {
		m.ParentID = 0
	}
	if err := h.svc.Update(c.Request.Context(), m); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, m, "更新成功")
}

// Delete DELETE /api/system/menu/:id —— 级联删除（自身 + 所有子孙）。
// @Summary      删除菜单（级联子孙）
// @Tags         menu
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "菜单 ID"
// @Success      200  {object} response.ApiResult
// @Failure      404  {object} response.ProblemDetail
// @Router       /system/menu/{id} [delete]
func (h *MenuHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), uint(id)); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, true, "删除成功")
}

// Sort PATCH /api/system/menu/sort —— 拖拽排序（inner/before/after）。
// @Summary      菜单拖拽排序
// @Description  按 inner/before/after 三种落点重排菜单层级与顺序
// @Tags         menu
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body service.MenuSortReq true "排序参数"
// @Success      200  {object} response.ApiResult
// @Router       /system/menu/sort [patch]
func (h *MenuHandler) Sort(c *gin.Context) {
	var req service.MenuSortReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	if err := h.svc.Sort(c.Request.Context(), &req); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, true, "排序成功")
}
