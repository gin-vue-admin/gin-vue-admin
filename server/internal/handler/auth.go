package handler

import (
	"github.com/gin-gonic/gin"
	"gva/internal/middleware"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/response"
	"gva/internal/service"
)

// LoginRequest 登录请求体。
type LoginRequest struct {
	// 登录仅校验非空：长度约束留给注册（M3）。
	// 在此加 min 长度会让短用户名在 service 之前返回 400，与用户不存在的 401 形成可区分信号，
	// 破坏纯防枚举（攻击者可推断用户名长度策略）。
	Username string `json:"username" binding:"required"`
	// bcrypt 72 字节上限是硬约束，密码校验不泄露用户名信息。
	Password string `json:"password" binding:"required,min=6,max=72"`
}

// RefreshRequest 刷新令牌请求体。
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// AuthHandler 认证端点。
type AuthHandler struct {
	svc *service.AuthService
}

// NewAuthHandler 构造认证处理器。
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Login POST /api/auth/sessions
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	res, err := h.svc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, res)
}

// Refresh POST /api/auth/tokens/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperr.Write(c, apperr.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	res, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, res)
}

// Logout DELETE /api/auth/sessions
// 纯 JWT 模式下为空操作，前端清本地 storage 即可。
func (h *AuthHandler) Logout(c *gin.Context) {
	_ = h.svc.Logout(c.Request.Context())
	response.Success(c, nil)
}

// Me GET /api/auth/users/me
// 从中间件注入的 userID 取当前用户档案。
func (h *AuthHandler) Me(c *gin.Context) {
	// 防御性断言：中间件应已注入 uint 类型 userID，缺失或类型异常一律视为未授权。
	uidAny, exists := c.Get(middleware.ContextKeyUserID)
	uid, ok := uidAny.(uint)
	if !exists || !ok {
		apperr.Write(c, apperr.Unauthorized("未授权"))
		return
	}
	prof, err := h.svc.GetProfile(c.Request.Context(), uid)
	if err != nil {
		apperr.Write(c, err)
		return
	}
	response.Success(c, prof)
}
