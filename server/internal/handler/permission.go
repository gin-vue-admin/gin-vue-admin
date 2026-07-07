// Package handler 权限管理端点：CRUD + 列表 + 导出。
// 依赖 service.PermissionService 与统一响应封装。
package handler

import (
	"strconv"

	"gva/internal/middleware"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/pagination"
	"gva/internal/pkg/response"
	"gva/internal/service"

	"github.com/gin-gonic/gin"
)

// permissionCreateReq 创建/更新共用请求体。
// status 用 oneof 约束 active/inactive，保持与种子数据一致。
type permissionCreateReq struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Module      string `json:"module" binding:"required"`
	Description string `json:"description"`
	Status      string `json:"status" binding:"required,oneof=active inactive"`
}

// PermissionHandler 权限端点处理器。
type PermissionHandler struct {
	svc *service.PermissionService
}

// NewPermissionHandler 构造权限处理器。
func NewPermissionHandler(svc *service.PermissionService) *PermissionHandler {
	return &PermissionHandler{svc: svc}
}

// List GET /api/permission
// 支持 page/size/keyword/status（走 pagination.Query）与 module（独立 query）。
// all=true/1 走 ListAll 返回全量数组，否则走分页。
// @Summary      权限列表
// @Tags         permission
// @Produce      json
// @Security     BearerAuth
// @Param        page    query int    false "页码"  default(1)
// @Param        size    query int    false "每页"  default(10)
// @Param        keyword query string false "名称/编码关键词"
// @Param        status  query string false "状态"  Enums(active, inactive)
// @Param        module  query string false "模块"
// @Param        all     query string false "all=true 返回全量"  Enums(true, 1)
// @Success      200  {object} response.ApiResult
// @Router       /permission [get]
func (h *PermissionHandler) List(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	module := c.Query("module")
	all := c.Query("all")
	if all == "true" || all == "1" {
		list, err := h.svc.ListAll(c.Request.Context(), q, module)
		if err != nil {
			apperr.Write(c, err)
			return
		}
		response.Success(c, list)
		return
	}
	res, err := h.svc.List(c.Request.Context(), q, module)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, res)
}

// Export GET /api/permission/export
// 返回 CSV 文本（含表头），仍受 module/status/keyword 过滤。
// @Summary      导出权限 CSV
// @Tags         permission
// @Produce      json
// @Security     BearerAuth
// @Param        keyword query string false "关键词"
// @Param        status  query string false "状态"
// @Param        module  query string false "模块"
// @Success      200  {object} response.ApiResult
// @Router       /permission/export [get]
func (h *PermissionHandler) Export(c *gin.Context) {
	var q pagination.Query
	_ = c.ShouldBindQuery(&q)
	module := c.Query("module")
	csv, err := h.svc.Export(c.Request.Context(), q, module)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, csv, "导出成功")
}

// Get GET /api/permission/:id
// @Summary      权限详情
// @Tags         permission
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "权限 ID"
// @Success      200  {object} response.ApiResult
// @Failure      404  {object} response.ProblemDetail
// @Router       /permission/{id} [get]
func (h *PermissionHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	p, err := h.svc.Get(c.Request.Context(), uint(id))
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, p)
}

// Create POST /api/permission
// @Summary      创建权限
// @Tags         permission
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body permissionCreateReq true "权限信息"
// @Success      200  {object} response.ApiResult
// @Failure      400  {object} response.ProblemDetail
// @Failure      409  {object} response.ProblemDetail "编码已存在"
// @Router       /permission [post]
func (h *PermissionHandler) Create(c *gin.Context) {
	var req permissionCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	p, err := h.svc.Create(c.Request.Context(), req.Name, req.Code, req.Module, req.Description, req.Status)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	// 权限新增后立即失效缓存，闭合越权窗口（旧码若被缓存会放行受保护路由）。
	middleware.InvalidateAll()
	response.Success(c, p, "创建成功")
}

// Update PUT /api/permission/:id
// 先查再改字段再更新：service.Update 接收完整实体。
// @Summary      更新权限
// @Tags         permission
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path int true "权限 ID"
// @Param        request body permissionCreateReq true "权限信息"
// @Success      200  {object} response.ApiResult
// @Failure      400  {object} response.ProblemDetail
// @Failure      404  {object} response.ProblemDetail
// @Router       /permission/{id} [put]
func (h *PermissionHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	var req permissionCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	// 先查再更新（service.Update 接收完整实体）
	p, err := h.svc.Get(c.Request.Context(), uint(id))
	if err != nil {
		apperr.Write(c, err)
		return
	}
	p.Name = req.Name
	p.Code = req.Code
	p.Module = req.Module
	p.Description = req.Description
	p.Status = req.Status
	if err := h.svc.Update(c.Request.Context(), p); err != nil {
		apperr.Write(c, err)
		return
	}
	// 权限更新后立即失效缓存，闭合越权窗口。
	middleware.InvalidateAll()
	response.Success(c, p, "更新成功")
}

// Delete DELETE /api/permission/:id
// @Summary      删除权限
// @Tags         permission
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "权限 ID"
// @Success      200  {object} response.ApiResult
// @Failure      404  {object} response.ProblemDetail
// @Router       /permission/{id} [delete]
func (h *PermissionHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), uint(id)); err != nil {
		apperr.Write(c, err)
		return
	}
	// 权限删除后立即失效缓存，闭合越权窗口（删权限后旧码仍命中受保护路由）。
	middleware.InvalidateAll()
	response.Success(c, true, "删除成功")
}

// BatchDelete DELETE /api/permission
// @Summary      批量删除权限
// @Tags         permission
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body batchDeleteReq true "ID 列表"
// @Success      200  {object} response.ApiResult
// @Failure      400  {object} response.ProblemDetail
// @Router       /permission [delete]
func (h *PermissionHandler) BatchDelete(c *gin.Context) {
	var req batchDeleteReq
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
	// 批量删除后立即失效缓存，闭合越权窗口。
	middleware.InvalidateAll()
	response.Success(c, true, "删除成功")
}
