package middleware

import (
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/audit"
	"gva/internal/pkg/jwt"

	"github.com/gin-gonic/gin"
)

// ContextKeyUserID / ContextKeyUsername 上下文键常量。
const (
	ContextKeyUserID   = "userID"
	ContextKeyUsername = "username"
)

// AuthRequired 校验 access token 并注入用户身份。仅 access token 可用，refresh 不行。
// 用户身份同时写入 gin context（c.Set，供 handler 取）与 request context
// （供 service/repo → GORM 审计回调自动写入 CreatedBy/UpdatedBy/DeletedBy）。
func AuthRequired(jwtMgr *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if len(auth) <= 7 || auth[:7] != "Bearer " {
			apperr.Write(c, apperr.Unauthorized("未授权"))
			return
		}
		tokenStr := auth[7:]
		claims, err := jwtMgr.Parse(tokenStr)
		if err != nil || claims.Type != jwt.TypeAccess {
			apperr.Write(c, apperr.Unauthorized("未授权"))
			return
		}
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUsername, claims.Username)
		// 注入 request context：下游 c.Request.Context() 携带 userID，GORM 审计回调据此写入
		c.Request = c.Request.WithContext(audit.WithUserID(c.Request.Context(), claims.UserID))
		c.Next()
	}
}
