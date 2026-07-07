// Package handler 操作日志端点：查询/详情/删除/批量删除/清空（无创建/编辑，日志由中间件生成）。
package handler

import (
	"strconv"

	"gva/internal/pkg/apperr"
	"gva/internal/pkg/response"
	"gva/internal/repository"
	"gva/internal/service"

	"github.com/gin-gonic/gin"
)

// opLogListQuery 操作日志查询参数（keyword/status/时间范围/page/size）。
type opLogListQuery struct {
	Keyword   string `form:"keyword"`
	Status    string `form:"status"`
	StartTime string `form:"startTime"`
	EndTime   string `form:"endTime"`
	Page      int    `form:"page"`
	Size      int    `form:"size"`
}

// opLogBatchDeleteReq 批量删除请求体。
type opLogBatchDeleteReq = batchDeleteReq

// OperationLogHandler 操作日志处理器。
type OperationLogHandler struct {
	svc *service.OperationLogService
}

func NewOperationLogHandler(svc *service.OperationLogService) *OperationLogHandler {
	return &OperationLogHandler{svc: svc}
}

// List GET /api/system/operation-log
// @Summary      操作日志分页列表
// @Tags         operation-log
// @Produce      json
// @Security     BearerAuth
// @Param        page      query int    false "页码" default(1)
// @Param        size      query int    false "每页" default(10)
// @Param        keyword   query string false "用户名/方法/路径关键词"
// @Param        status    query string false "状态码"
// @Param        startTime query string false "起始时间"
// @Param        endTime   query string false "结束时间"
// @Success      200  {object} response.ApiResult
// @Router       /system/operation-log [get]
func (h *OperationLogHandler) List(c *gin.Context) {
	var q opLogListQuery
	_ = c.ShouldBindQuery(&q)
	oq := repository.OperationLogQuery{StartTime: q.StartTime, EndTime: q.EndTime}
	oq.Keyword, oq.Status, oq.Page, oq.Size = q.Keyword, q.Status, q.Page, q.Size
	res, err := h.svc.List(c.Request.Context(), oq)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, res)
}

// Get GET /api/system/operation-log/:id
// @Summary      操作日志详情
// @Tags         operation-log
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "ID"
// @Success      200  {object} response.ApiResult
// @Failure      404  {object} response.ProblemDetail
// @Router       /system/operation-log/{id} [get]
func (h *OperationLogHandler) Get(c *gin.Context) {
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

// Delete DELETE /api/system/operation-log/:id
// @Summary      删除操作日志
// @Tags         operation-log
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "ID"
// @Success      200  {object} response.ApiResult
// @Router       /system/operation-log/{id} [delete]
func (h *OperationLogHandler) Delete(c *gin.Context) {
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

// BatchDelete DELETE /api/system/operation-log
// @Summary      批量删除操作日志
// @Tags         operation-log
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body opLogBatchDeleteReq true "ID 列表"
// @Success      200  {object} response.ApiResult
// @Router       /system/operation-log [delete]
func (h *OperationLogHandler) BatchDelete(c *gin.Context) {
	var req opLogBatchDeleteReq
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

// Clear DELETE /api/system/operation-log/clear  清空全部。
// @Summary      清空操作日志
// @Tags         operation-log
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} response.ApiResult
// @Router       /system/operation-log/clear [delete]
func (h *OperationLogHandler) Clear(c *gin.Context) {
	if err := h.svc.Clear(c.Request.Context()); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, true, "清空成功")
}
