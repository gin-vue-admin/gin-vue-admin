package server

import (
	"github.com/gin-gonic/gin"
	"gva/internal/handler"
	"gva/internal/middleware"
	"gva/internal/pkg/jwt"
	"gva/internal/pkg/response"
)

// NewRouter 装配全局中间件与路由。
func NewRouter(authHandler *handler.AuthHandler, jwtMgr *jwt.Manager) *gin.Engine {
	r := gin.New()
	r.HandleMethodNotAllowed = true
	r.Use(
		middleware.TraceID(),
		middleware.Recovery(),
		middleware.Logger(),
		middleware.CORS(),
	)

	// 统一 404/405 为 ProblemDetail，保持响应契约一致。
	r.NoRoute(func(c *gin.Context) {
		response.Problem(c, 404, "", "资源不存在")
	})
	r.NoMethod(func(c *gin.Context) {
		response.Problem(c, 405, "", "方法不允许")
	})

	api := r.Group("/api")
	{
		health := handler.NewHealthHandler()
		api.GET("/health", health.Health)

		// 认证
		auth := api.Group("/auth")
		{
			auth.POST("/sessions", authHandler.Login)
			auth.DELETE("/sessions", authHandler.Logout)
			auth.POST("/tokens/refresh", authHandler.Refresh)
			// me 需 access token 校验
			auth.GET("/users/me", middleware.AuthRequired(jwtMgr), authHandler.Me)
		}

		// TODO(M3): /api/user /api/role /api/permission
		// TODO(M4): /api/system/menus
	}

	return r
}
