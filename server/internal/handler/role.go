// Package handler 角色管理端点：CRUD + 列表 + 导出 + 权限分配。
// 依赖 service.RoleService 与统一响应封装。
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/pagination"
	"gva/internal/pkg/response"
	"gva/internal/service"
)

// roleCreateReq 创建/更新角色共用请求体。
// status 用 oneof 约束 active/inactive，保持与种子数据一致。
// description 限制最大长度 255，与 model.Role 的 gorm size:255 对齐。
type roleCreateReq struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description" binding:"max=255"`
	Status      string `json:"status" binding:"required,oneof=active inactive"`
}

// setPermissionsReq 角色权限分配请求体，permissions 为权限 code 数组。
type setPermissionsReq struct {
	Permissions []string `json:"permissions" binding:"required"`
}

// roleBatchDeleteReq 批量删除请求体，ids 为字符串数组（前端契约）。
// 注：与 permission handler 的 batchDeleteReq 同包冲突，故加 role 前缀。
type roleBatchDeleteReq struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// RoleHandler 角色端点处理器。
type RoleHandler struct {
	svc *service.RoleService
}

// NewRoleHandler 构造角色处理器。
func NewRoleHandler(svc *service.RoleService) *RoleHandler {
	return &RoleHandler{svc: svc}
}

// List GET /api/role
// 支持 page/size/keyword/status（走 pagination.Query）。
func (h *RoleHandler) List(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	res, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, res)
}

// Export GET /api/role/export
// 返回 CSV 文本（含表头），仍受 keyword/status 过滤。
func (h *RoleHandler) Export(c *gin.Context) {
	var q pagination.Query
	_ = c.ShouldBindQuery(&q)
	csv, err := h.svc.Export(c.Request.Context(), q)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, csv, "导出成功")
}

// Get GET /api/role/:id
func (h *RoleHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	r, err := h.svc.Get(c.Request.Context(), uint(id))
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, r)
}

// Create POST /api/role
func (h *RoleHandler) Create(c *gin.Context) {
	var req roleCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	r, err := h.svc.Create(c.Request.Context(), req.Name, req.Code, req.Description, req.Status)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, r, "创建成功")
}

// Update PUT /api/role/:id
// 先查再改字段再更新：service.Update 接收完整实体。
func (h *RoleHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	var req roleCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	r, err := h.svc.Get(c.Request.Context(), uint(id))
	if err != nil {
		apperr.Write(c, err)
		return
	}
	r.Name = req.Name
	r.Code = req.Code
	r.Description = req.Description
	r.Status = req.Status
	if err := h.svc.Update(c.Request.Context(), r); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, r, "更新成功")
}

// Delete DELETE /api/role/:id
func (h *RoleHandler) Delete(c *gin.Context) {
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

// BatchDelete DELETE /api/role
func (h *RoleHandler) BatchDelete(c *gin.Context) {
	var req roleBatchDeleteReq
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

// GetPermissions GET /api/role/:id/permissions
// 返回当前角色已分配的权限 code 数组。
func (h *RoleHandler) GetPermissions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	codes, err := h.svc.GetPermissions(c.Request.Context(), uint(id))
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, codes)
}

// SetPermissions PUT /api/role/:id/permissions
// 全量覆盖角色的权限集合（增量语义由 service 处理）。
// service.SetPermissions 已在成功后调 middleware.InvalidateAll() 失效权限缓存。
func (h *RoleHandler) SetPermissions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	var req setPermissionsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	if err := h.svc.SetPermissions(c.Request.Context(), uint(id), req.Permissions); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, true, "设置成功")
}
