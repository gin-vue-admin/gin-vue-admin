package server

import (
	"github.com/gin-gonic/gin"
	"gva/internal/handler"
	"gva/internal/middleware"
	"gva/internal/pkg/jwt"
	"gva/internal/pkg/response"
	"gva/internal/repository"
)

// NewRouter 装配全局中间件与路由。
// permRepo 传入路由层仅用于 RequirePermission 中间件（按 userID 查权限码集合）。
func NewRouter(authHandler *handler.AuthHandler, permHandler *handler.PermissionHandler, permRepo repository.PermissionRepository, jwtMgr *jwt.Manager) *gin.Engine {
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

		// 权限管理
		perm := api.Group("/permission")
		perm.Use(middleware.AuthRequired(jwtMgr))
		{
			// export 必须在 :id 之前注册（Gin 树形路由精确匹配 export 优先）
			perm.GET("/export", middleware.RequirePermission(permRepo, "permission:list"), permHandler.Export)
			perm.GET("", middleware.RequirePermission(permRepo, "permission:list"), permHandler.List)
			perm.GET("/:id", middleware.RequirePermission(permRepo, "permission:list"), permHandler.Get)
			perm.POST("", middleware.RequirePermission(permRepo, "permission:create"), permHandler.Create)
			perm.PUT("/:id", middleware.RequirePermission(permRepo, "permission:edit"), permHandler.Update)
			perm.DELETE("/:id", middleware.RequirePermission(permRepo, "permission:delete"), permHandler.Delete)
			perm.DELETE("", middleware.RequirePermission(permRepo, "permission:delete"), permHandler.BatchDelete)
		}

		// TODO(M3): /api/user /api/role
		// TODO(M4): /api/system/menus
	}

	return r
}
