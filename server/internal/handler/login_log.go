// Package handler 登录日志端点：查询/详情/删除/批量删除/清空（无创建/编辑，日志由 AuthService 生成）。
package handler

import (
	"strconv"

	"gva/internal/pkg/apperr"
	"gva/internal/pkg/response"
	"gva/internal/repository"
	"gva/internal/service"

	"github.com/gin-gonic/gin"
)

// loginLogListQuery 登录日志查询参数（keyword/status/时间范围/page/size）。
type loginLogListQuery struct {
	Keyword   string `form:"keyword"`
	Status    string `form:"status"`
	StartTime string `form:"startTime"`
	EndTime   string `form:"endTime"`
	Page      int    `form:"page"`
	Size      int    `form:"size"`
}

// loginLogBatchDeleteReq 批量删除请求体。
type loginLogBatchDeleteReq struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// LoginLogHandler 登录日志处理器。
type LoginLogHandler struct {
	svc *service.LoginLogService
}

func NewLoginLogHandler(svc *service.LoginLogService) *LoginLogHandler {
	return &LoginLogHandler{svc: svc}
}

// List GET /api/system/login-log
// @Summary      登录日志分页列表
// @Tags         login-log
// @Produce      json
// @Security     BearerAuth
// @Param        page      query int    false "页码" default(1)
// @Param        size      query int    false "每页" default(10)
// @Param        keyword   query string false "用户名/IP 关键词"
// @Param        status    query string false "success/failed"
// @Param        startTime query string false "起始时间"
// @Param        endTime   query string false "结束时间"
// @Success      200  {object} response.ApiResult
// @Router       /system/login-log [get]
func (h *LoginLogHandler) List(c *gin.Context) {
	var q loginLogListQuery
	_ = c.ShouldBindQuery(&q)
	lq := repository.LoginLogQuery{StartTime: q.StartTime, EndTime: q.EndTime}
	lq.Keyword, lq.Status, lq.Page, lq.Size = q.Keyword, q.Status, q.Page, q.Size
	res, err := h.svc.List(c.Request.Context(), lq)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, res)
}

// Get GET /api/system/login-log/:id
// @Summary      登录日志详情
// @Tags         login-log
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "ID"
// @Success      200  {object} response.ApiResult
// @Failure      404  {object} response.ProblemDetail
// @Router       /system/login-log/{id} [get]
func (h *LoginLogHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	e, err := h.svc.Get(c.Request.Context(), uint(id))
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, e)
}

// Delete DELETE /api/system/login-log/:id
// @Summary      删除登录日志
// @Tags         login-log
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "ID"
// @Success      200  {object} response.ApiResult
// @Router       /system/login-log/{id} [delete]
func (h *LoginLogHandler) Delete(c *gin.Context) {
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

// BatchDelete DELETE /api/system/login-log
// @Summary      批量删除登录日志
// @Tags         login-log
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body loginLogBatchDeleteReq true "ID 列表"
// @Success      200  {object} response.ApiResult
// @Router       /system/login-log [delete]
func (h *LoginLogHandler) BatchDelete(c *gin.Context) {
	var req loginLogBatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	ids := make([]uint, 0, len(req.IDs))
	for _, s := range req.IDs {
		id, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			apperr.Write(c, apperr.Validation("无效的 id: "+s, nil))
			return
		}
		ids = append(ids, uint(id))
	}
	if err := h.svc.BatchDelete(c.Request.Context(), ids); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, true, "删除成功")
}

// Clear DELETE /api/system/login-log/clear  清空全部。
// @Summary      清空登录日志
// @Tags         login-log
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} response.ApiResult
// @Router       /system/login-log/clear [delete]
func (h *LoginLogHandler) Clear(c *gin.Context) {
	if err := h.svc.Clear(c.Request.Context()); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, true, "清空成功")
}
