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
// 受数据范围约束：按当前用户角色 DataScope 过滤可见用户。
// @Summary      用户分页列表
// @Tags         user
// @Produce      json
// @Security     BearerAuth
// @Param        page      query int    false "页码"   default(1)
// @Param        size      query int    false "每页"   default(10)
// @Param        keyword   query string false "用户名/姓名/手机号关键词"
// @Param        status    query string false "状态"   Enums(active, inactive)
// @Param        role      query string false "角色编码过滤"
// @Success      200  {object} response.ApiResult
// @Failure      401  {object} response.ProblemDetail
// @Failure      403  {object} response.ProblemDetail
// @Router       /user [get]
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
// @Summary      导出用户 CSV
// @Tags         user
// @Produce      json
// @Security     BearerAuth
// @Param        keyword query string false "关键词"
// @Param        status  query string false "状态"
// @Param        role    query string false "角色编码"
// @Success      200  {object} response.ApiResult
// @Router       /user/export [get]
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
// @Summary      用户详情
// @Tags         user
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "用户 ID"
// @Success      200  {object} response.ApiResult
// @Failure      400  {object} response.ProblemDetail
// @Failure      404  {object} response.ProblemDetail
// @Router       /user/{id} [get]
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
// @Summary      创建用户
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body service.UserCreateReq true "用户信息"
// @Success      200  {object} response.ApiResult
// @Failure      400  {object} response.ProblemDetail
// @Failure      404  {object} response.ProblemDetail "角色不存在"
// @Failure      409  {object} response.ProblemDetail "用户名已存在"
// @Router       /user [post]
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
// @Summary      更新用户
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path int true "用户 ID"
// @Param        request body service.UserUpdateReq true "用户更新字段（指针字段 nil 不改）"
// @Success      200  {object} response.ApiResult
// @Failure      400  {object} response.ProblemDetail
// @Failure      404  {object} response.ProblemDetail
// @Failure      409  {object} response.ProblemDetail "禁用/删除自己"
// @Router       /user/{id} [put]
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
// @Summary      删除用户（软删）
// @Tags         user
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "用户 ID"
// @Success      200  {object} response.ApiResult
// @Failure      404  {object} response.ProblemDetail
// @Failure      409  {object} response.ProblemDetail "删除自己"
// @Router       /user/{id} [delete]
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
// @Summary      批量删除用户
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body userBatchDeleteReq true "用户 ID 列表"
// @Success      200  {object} response.ApiResult
// @Failure      400  {object} response.ProblemDetail
// @Failure      409  {object} response.ProblemDetail "含自己"
// @Router       /user [delete]
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
