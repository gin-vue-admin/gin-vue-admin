package handler

import (
	"github.com/gin-gonic/gin"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/response"
	"gva/internal/service"
)

// MenuHandler 菜单端点。
type MenuHandler struct {
	svc *service.MenuService
}

func NewMenuHandler(svc *service.MenuService) *MenuHandler {
	return &MenuHandler{svc: svc}
}

// Menus GET /api/system/menus —— 当前用户菜单树（后端下发完整树，前端按权限过滤）。
func (h *MenuHandler) Menus(c *gin.Context) {
	tree, err := h.svc.GetMenus(c.Request.Context())
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, tree)
}
