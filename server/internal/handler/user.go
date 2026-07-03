package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gva/internal/middleware"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/pagination"
	"gva/internal/pkg/response"
	"gva/internal/service"
)

// userBatchDeleteReq 批量删除请求；命名加 user 前缀避免与 permission/role 同包冲突。
type userBatchDeleteReq struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// UserHandler 用户管理 HTTP handler。
type UserHandler struct {
	svc *service.UserService
}

// NewUserHandler 构造用户 handler。
func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// operatorID 从 context 取当前用户 id（AuthRequired 注入），用于防自删/自禁。
func operatorID(c *gin.Context) uint {
	if v, ok := c.Get(middleware.ContextKeyUserID); ok {
		if uid, ok := v.(uint); ok {
			return uid
		}
	}
	// AuthRequired 保证注入 userID；返回 0 表示异常（中间件漏挂），
	// 此时防自删失效但不致误操作（0 不等于任何有效 id）。
	return 0
}

// List GET /api/user 分页列表（含 roles 数组）。
func (h *UserHandler) List(c *gin.Context) {
	var q pagination.Query
	if err := c.ShouldBindQuery(&q); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	roleCode := c.Query("role")
	res, err := h.svc.List(c.Request.Context(), q, roleCode)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, res)
}

// Export GET /api/user/export 导出 CSV。
func (h *UserHandler) Export(c *gin.Context) {
	var q pagination.Query
	_ = c.ShouldBindQuery(&q)
	roleCode := c.Query("role")
	csv, err := h.svc.Export(c.Request.Context(), q, roleCode)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, csv, "导出成功")
}

// Get GET /api/user/:id 详情。
func (h *UserHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	u, err := h.svc.Get(c.Request.Context(), uint(id))
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, u)
}

// Create POST /api/user 创建用户（含密码哈希+角色绑定）。
func (h *UserHandler) Create(c *gin.Context) {
	var req service.UserCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	u, err := h.svc.Create(c.Request.Context(), req.Username, req.RealName, req.Email, req.Phone, req.Roles, req.Status, req.Password)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, u, "创建成功")
}

// Update PUT /api/user/:id 更新（指针字段 nil 不改；传 operatorID 防自禁）。
func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	var req service.UserUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.Validation(err.Error(), nil))
		return
	}
	u, err := h.svc.Update(c.Request.Context(), uint(id), operatorID(c), &req)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, u, "更新成功")
}

// Delete DELETE /api/user/:id 软删（传 operatorID 防自删）。
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		apperr.Write(c, apperr.Validation("无效的 id", nil))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), uint(id), operatorID(c)); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, true, "删除成功")
}

// BatchDelete DELETE /api/user 批量软删（任一 id==operatorID→409）。
func (h *UserHandler) BatchDelete(c *gin.Context) {
	var req userBatchDeleteReq
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
	if err := h.svc.BatchDelete(c.Request.Context(), ids, operatorID(c)); err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, true, "删除成功")
}
