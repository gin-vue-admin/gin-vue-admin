// Package handler dashboard 首页统计端点。
package handler

import (
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/response"
	"gva/internal/service"

	"github.com/gin-gonic/gin"
)

// DashboardHandler 首页统计处理器。
type DashboardHandler struct {
	svc *service.DashboardService
}

// NewDashboardHandler 构造首页统计处理器。
func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

// chartRange 天数映射：7d/30d/90d → 7/30/90，缺省 7。
func chartDays(rangeStr string) int {
	switch rangeStr {
	case "30d":
		return 30
	case "90d":
		return 90
	default:
		return 7
	}
}

// Stats GET /api/dashboard/stats
// @Summary      首页统计卡片
// @Tags         dashboard
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} response.ApiResult
// @Router       /dashboard/stats [get]
func (h *DashboardHandler) Stats(c *gin.Context) {
	res, err := h.svc.Stats(c.Request.Context())
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, res)
}

// Charts GET /api/dashboard/charts?range=7d|30d|90d
// @Summary      首页图表（趋势 + 分布）
// @Tags         dashboard
// @Produce      json
// @Security     BearerAuth
// @Param        range query string false "7d/30d/90d" default(7d)
// @Success      200  {object} response.ApiResult
// @Router       /dashboard/charts [get]
func (h *DashboardHandler) Charts(c *gin.Context) {
	res, err := h.svc.Charts(c.Request.Context(), chartDays(c.Query("range")))
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, res)
}

// Activities GET /api/dashboard/activities
// @Summary      最近活动
// @Tags         dashboard
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} response.ApiResult
// @Router       /dashboard/activities [get]
func (h *DashboardHandler) Activities(c *gin.Context) {
	res, err := h.svc.Activities(c.Request.Context())
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, res)
}
