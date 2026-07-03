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
func NewRouter(authHandler *handler.AuthHandler, permHandler *handler.PermissionHandler, roleHandler *handler.RoleHandler, userHandler *handler.UserHandler, permRepo repository.PermissionRepository, jwtMgr *jwt.Manager) *gin.Engine {
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

		// 角色管理
		role := api.Group("/role")
		role.Use(middleware.AuthRequired(jwtMgr))
		{
			// 路由顺序：/export、/:id/permissions 必须在 /:id 前注册，
			// 确保 Gin 树形路由按静态/子路径优先匹配（避免被 /:id 抢占）。
			role.GET("/export", middleware.RequirePermission(permRepo, "role:list"), roleHandler.Export)
			role.GET("/:id/permissions", middleware.RequirePermission(permRepo, "role:permission"), roleHandler.GetPermissions)
			role.PUT("/:id/permissions", middleware.RequirePermission(permRepo, "role:permission"), roleHandler.SetPermissions)
			role.GET("", middleware.RequirePermission(permRepo, "role:list"), roleHandler.List)
			role.GET("/:id", middleware.RequirePermission(permRepo, "role:list"), roleHandler.Get)
			role.POST("", middleware.RequirePermission(permRepo, "role:create"), roleHandler.Create)
			role.PUT("/:id", middleware.RequirePermission(permRepo, "role:edit"), roleHandler.Update)
			role.DELETE("/:id", middleware.RequirePermission(permRepo, "role:delete"), roleHandler.Delete)
			role.DELETE("", middleware.RequirePermission(permRepo, "role:delete"), roleHandler.BatchDelete)
		}

		// 用户管理
		user := api.Group("/user")
		user.Use(middleware.AuthRequired(jwtMgr))
		{
			// 路由顺序：/export 必须在 /:id 前注册，确保 Gin 树形路由按静态路径优先匹配。
			user.GET("/export", middleware.RequirePermission(permRepo, "user:list"), userHandler.Export)
			user.GET("", middleware.RequirePermission(permRepo, "user:list"), userHandler.List)
			user.GET("/:id", middleware.RequirePermission(permRepo, "user:list"), userHandler.Get)
			user.POST("", middleware.RequirePermission(permRepo, "user:create"), userHandler.Create)
			user.PUT("/:id", middleware.RequirePermission(permRepo, "user:edit"), userHandler.Update)
			user.DELETE("/:id", middleware.RequirePermission(permRepo, "user:delete"), userHandler.Delete)
			user.DELETE("", middleware.RequirePermission(permRepo, "user:delete"), userHandler.BatchDelete)
		}

		// TODO(M4): /api/system/menus
	}

	return r
}
