// Package handler notice 公告端点。
package handler

import (
	"strconv"

	"gva/internal/pkg/apperr"
	"gva/internal/pkg/pagination"
	"gva/internal/pkg/response"
	"gva/internal/service"

	"github.com/gin-gonic/gin"
)

// noticeBatchDeleteReq 批量删除请求。
type noticeBatchDeleteReq struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// NoticeHandler 公告处理器。
type NoticeHandler struct {
	svc *service.NoticeService
}

// NewNoticeHandler 构造公告处理器。
func NewNoticeHandler(svc *service.NoticeService) *NoticeHandler {
	return &NoticeHandler{svc: svc}
}

// List GET /api/system/notice
// @Summary      公告分页列表
// @Tags         notice
// @Produce      json
// @Security     BearerAuth
// @Param        page    query int    false "页码" default(1)
// @Param        size    query int    false "每页" default(10)
// @Param        keyword query string false "标题关键词"
// @Param        type    query string false "announcement/notice/todo"
// @Param        status  query string false "published/draft/expired"
// @Success      200  {object} response.ApiResult
// @Router       /system/notice [get]
func (h *NoticeHandler) List(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	res, err := h.svc.List(c.Request.Context(), q, c.Query("type"))
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, res)
}

// Get GET /api/system/notice/:id
// @Summary      公告详情
// @Tags         notice
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "ID"
// @Success      200  {object} response.ApiResult
// @Failure      404  {object} response.ProblemDetail
// @Router       /system/notice/{id} [get]
func (h *NoticeHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	n, err := h.svc.Get(c.Request.Context(), uint(id))
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, n)
}

// Create POST /api/system/notice
// @Summary      创建公告
// @Tags         notice
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body service.NoticeUpsertReq true "公告信息"
// @Success      200  {object} response.ApiResult
// @Router       /system/notice [post]
func (h *NoticeHandler) Create(c *gin.Context) {
	var req service.NoticeUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	n, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, n, "创建成功")
}

// Update PUT /api/system/notice/:id
// @Summary      更新公告
// @Tags         notice
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path int true "ID"
// @Param        request body service.NoticeUpsertReq true "公告信息"
// @Success      200  {object} response.ApiResult
// @Router       /system/notice/{id} [put]
func (h *NoticeHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	var req service.NoticeUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	n, err := h.svc.Update(c.Request.Context(), uint(id), &req)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, n, "更新成功")
}

// Delete DELETE /api/system/notice/:id
// @Summary      删除公告
// @Tags         notice
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "ID"
// @Success      200  {object} response.ApiResult
// @Router       /system/notice/{id} [delete]
func (h *NoticeHandler) Delete(c *gin.Context) {
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

// BatchDelete DELETE /api/system/notice
// @Summary      批量删除公告
// @Tags         notice
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body noticeBatchDeleteReq true "ID 列表"
// @Success      200  {object} response.ApiResult
// @Router       /system/notice [delete]
func (h *NoticeHandler) BatchDelete(c *gin.Context) {
	var req noticeBatchDeleteReq
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

// Publish POST /api/system/notice/:id/publish
// @Summary      发布公告
// @Tags         notice
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "ID"
// @Success      200  {object} response.ApiResult
// @Router       /system/notice/{id}/publish [post]
func (h *NoticeHandler) Publish(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	if err := h.svc.Publish(c.Request.Context(), uint(id)); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, true, "发布成功")
}

// Revoke POST /api/system/notice/:id/revoke
// @Summary      撤销公告
// @Tags         notice
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "ID"
// @Success      200  {object} response.ApiResult
// @Router       /system/notice/{id}/revoke [post]
func (h *NoticeHandler) Revoke(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	if err := h.svc.Revoke(c.Request.Context(), uint(id)); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, true, "撤销成功")
}

// Export GET /api/system/notice/export
// @Summary      导出公告 CSV
// @Tags         notice
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} response.ApiResult
// @Router       /system/notice/export [get]
func (h *NoticeHandler) Export(c *gin.Context) {
	csv, err := h.svc.Export(c.Request.Context())
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, csv, "导出成功")
}
