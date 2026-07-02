package middleware

import (
	"github.com/gin-gonic/gin"
	"gva/internal/pkg/apperr"
	"gva/internal/pkg/jwt"
)

// ContextKeyUserID / ContextKeyUsername 上下文键常量。
const (
	ContextKeyUserID   = "userID"
	ContextKeyUsername = "username"
)

// AuthRequired 校验 access token 并注入用户身份。仅 access token 可用，refresh 不行。
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
		c.Next()
	}
}
