package server

import (
	"gva/internal/handler"
	"gva/internal/middleware"
	"gva/internal/pkg/async"
	"gva/internal/pkg/jwt"
	"gva/internal/pkg/response"
	"gva/internal/repository"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// dictCRUDHandler 字典三级 handler 共同满足的接口，用于 registerDictRoutes 消除重复注册。
type dictCRUDHandler interface {
	Export(c *gin.Context)
	List(c *gin.Context)
	Get(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
	BatchDelete(c *gin.Context)
}

func registerDictRoutes(g *gin.RouterGroup, permRepo repository.PermissionRepository, code string, h dictCRUDHandler) {
	g.GET("/export", middleware.RequirePermission(permRepo, code+":list"), h.Export)
	g.GET("", middleware.RequirePermission(permRepo, code+":list"), h.List)
	g.GET("/:id", middleware.RequirePermission(permRepo, code+":list"), h.Get)
	g.POST("", middleware.RequirePermission(permRepo, code+":create"), h.Create)
	g.PUT("/:id", middleware.RequirePermission(permRepo, code+":edit"), h.Update)
	g.DELETE("/:id", middleware.RequirePermission(permRepo, code+":delete"), h.Delete)
	g.DELETE("", middleware.RequirePermission(permRepo, code+":delete"), h.BatchDelete)
}

// NewRouter 装配全局中间件与路由。
func NewRouter(authHandler *handler.AuthHandler, permHandler *handler.PermissionHandler, roleHandler *handler.RoleHandler, userHandler *handler.UserHandler, menuHandler *handler.MenuHandler, crudHandler *handler.CrudHandler, deptHandler *handler.DeptHandler, dictCategoryHandler *handler.DictCategoryHandler, dictHandler *handler.DictHandler, dictItemHandler *handler.DictItemHandler, opLogHandler *handler.OperationLogHandler, loginLogHandler *handler.LoginLogHandler, sysConfigHandler *handler.SysConfigHandler, dashboardHandler *handler.DashboardHandler, noticeHandler *handler.NoticeHandler, permRepo repository.PermissionRepository, opLogRepo repository.OperationLogRepository, jwtMgr *jwt.Manager, runner async.Runner) *gin.Engine {
	r := gin.New()
	r.HandleMethodNotAllowed = true
	r.Use(
		middleware.TraceID(),
		middleware.Recovery(),
		middleware.Logger(),
		middleware.CORS(),
	)

	r.NoRoute(func(c *gin.Context) { response.Problem(c, 404, "", "资源不存在") })
	r.NoMethod(func(c *gin.Context) { response.Problem(c, 405, "", "方法不允许") })

	// M8 Swagger UI：访问 /swagger/index.html 查看 API 文档（无需鉴权）。
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api")
	// 操作日志中间件：记录所有写操作（GET 跳过）。挂在 api 组，username 从 AuthRequired 注入取。
	api.Use(middleware.OperationLog(opLogRepo, runner))
	{
		health := handler.NewHealthHandler()
		api.GET("/health", health.Health)

		// 公开配置（无需鉴权，前端启动拉取 site_title/login_captcha 等）
		api.GET("/system/config/public", sysConfigHandler.GetPublic)

		// 首页统计（登录即可见，无权限码限制）
		dashboard := api.Group("/dashboard")
		dashboard.Use(middleware.AuthRequired(jwtMgr))
		{
			dashboard.GET("/stats", dashboardHandler.Stats)
			dashboard.GET("/charts", dashboardHandler.Charts)
			dashboard.GET("/activities", dashboardHandler.Activities)
		}

		auth := api.Group("/auth")
		{
			auth.POST("/sessions", authHandler.Login)
			auth.DELETE("/sessions", authHandler.Logout)
			auth.POST("/tokens/refresh", authHandler.Refresh)
			auth.GET("/users/me", middleware.AuthRequired(jwtMgr), authHandler.Me)
		}

		perm := api.Group("/permission")
		perm.Use(middleware.AuthRequired(jwtMgr))
		{
			perm.GET("/export", middleware.RequirePermission(permRepo, "permission:list"), permHandler.Export)
			perm.GET("", middleware.RequirePermission(permRepo, "permission:list"), permHandler.List)
			perm.GET("/:id", middleware.RequirePermission(permRepo, "permission:list"), permHandler.Get)
			perm.POST("", middleware.RequirePermission(permRepo, "permission:create"), permHandler.Create)
			perm.PUT("/:id", middleware.RequirePermission(permRepo, "permission:edit"), permHandler.Update)
			perm.DELETE("/:id", middleware.RequirePermission(permRepo, "permission:delete"), permHandler.Delete)
			perm.DELETE("", middleware.RequirePermission(permRepo, "permission:delete"), permHandler.BatchDelete)
		}

		role := api.Group("/role")
		role.Use(middleware.AuthRequired(jwtMgr))
		{
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

		user := api.Group("/user")
		user.Use(middleware.AuthRequired(jwtMgr))
		{
			user.GET("/export", middleware.RequirePermission(permRepo, "user:list"), userHandler.Export)
			user.GET("", middleware.RequirePermission(permRepo, "user:list"), userHandler.List)
			user.GET("/:id", middleware.RequirePermission(permRepo, "user:list"), userHandler.Get)
			user.POST("", middleware.RequirePermission(permRepo, "user:create"), userHandler.Create)
			user.PUT("/:id", middleware.RequirePermission(permRepo, "user:edit"), userHandler.Update)
			user.DELETE("/:id", middleware.RequirePermission(permRepo, "user:delete"), userHandler.Delete)
			user.DELETE("", middleware.RequirePermission(permRepo, "user:delete"), userHandler.BatchDelete)
		}

		crud := api.Group("/crud")
		crud.Use(middleware.AuthRequired(jwtMgr))
		{
			crud.GET("", middleware.RequirePermission(permRepo, "crud:list"), crudHandler.List)
			crud.GET("/:id", middleware.RequirePermission(permRepo, "crud:list"), crudHandler.Get)
			crud.POST("", middleware.RequirePermission(permRepo, "crud:create"), crudHandler.Create)
			crud.PUT("/:id", middleware.RequirePermission(permRepo, "crud:edit"), crudHandler.Update)
			crud.DELETE("/:id", middleware.RequirePermission(permRepo, "crud:delete"), crudHandler.Delete)
			crud.DELETE("", middleware.RequirePermission(permRepo, "crud:delete"), crudHandler.BatchDelete)
		}

		dict := api.Group("/dict")
		dict.Use(middleware.AuthRequired(jwtMgr))
		registerDictRoutes(dict.Group("/categories"), permRepo, "dict", dictCategoryHandler)
		registerDictRoutes(dict.Group("/dicts"), permRepo, "dict", dictHandler)
		registerDictRoutes(dict.Group("/items"), permRepo, "dict", dictItemHandler)

		sys := api.Group("/system")
		sys.Use(middleware.AuthRequired(jwtMgr))
		{
			sys.GET("/menus", menuHandler.Menus)

			menu := sys.Group("/menu")
			{
				menu.GET("", middleware.RequirePermission(permRepo, "menu:list"), menuHandler.GetTree)
				menu.POST("", middleware.RequirePermission(permRepo, "menu:create"), menuHandler.Create)
				menu.PUT("/:id", middleware.RequirePermission(permRepo, "menu:edit"), menuHandler.Update)
				menu.DELETE("/:id", middleware.RequirePermission(permRepo, "menu:delete"), menuHandler.Delete)
				menu.PATCH("/sort", middleware.RequirePermission(permRepo, "menu:edit"), menuHandler.Sort)
			}

			dept := sys.Group("/dept")
			{
				dept.GET("/export", middleware.RequirePermission(permRepo, "dept:list"), deptHandler.Export)
				dept.GET("", middleware.RequirePermission(permRepo, "dept:list"), deptHandler.List)
				dept.GET("/:id", middleware.RequirePermission(permRepo, "dept:list"), deptHandler.Get)
				dept.POST("", middleware.RequirePermission(permRepo, "dept:create"), deptHandler.Create)
				dept.PUT("/:id", middleware.RequirePermission(permRepo, "dept:edit"), deptHandler.Update)
				dept.DELETE("/:id", middleware.RequirePermission(permRepo, "dept:delete"), deptHandler.Delete)
				dept.DELETE("", middleware.RequirePermission(permRepo, "dept:delete"), deptHandler.BatchDelete)
			}

			// 操作日志（路由顺序：/clear 在 /:id 前注册）
			opLog := sys.Group("/operation-log")
			{
				opLog.GET("", middleware.RequirePermission(permRepo, "system:log"), opLogHandler.List)
				opLog.DELETE("/clear", middleware.RequirePermission(permRepo, "system:log"), opLogHandler.Clear)
				opLog.DELETE("", middleware.RequirePermission(permRepo, "system:log"), opLogHandler.BatchDelete)
				opLog.GET("/:id", middleware.RequirePermission(permRepo, "system:log"), opLogHandler.Get)
				opLog.DELETE("/:id", middleware.RequirePermission(permRepo, "system:log"), opLogHandler.Delete)
			}

			// 登录日志（复用 system:log 权限码；路由顺序：/clear 在 /:id 前注册）
			loginLog := sys.Group("/login-log")
			{
				loginLog.GET("", middleware.RequirePermission(permRepo, "system:log"), loginLogHandler.List)
				loginLog.DELETE("/clear", middleware.RequirePermission(permRepo, "system:log"), loginLogHandler.Clear)
				loginLog.DELETE("", middleware.RequirePermission(permRepo, "system:log"), loginLogHandler.BatchDelete)
				loginLog.GET("/:id", middleware.RequirePermission(permRepo, "system:log"), loginLogHandler.Get)
				loginLog.DELETE("/:id", middleware.RequirePermission(permRepo, "system:log"), loginLogHandler.Delete)
			}

			// 系统参数配置（路由顺序：/key/:key 静态段优先于 /:id，Gin radix 自动区分）
			sysCfg := sys.Group("/config")
			{
				sysCfg.GET("", middleware.RequirePermission(permRepo, "config:system"), sysConfigHandler.List)
				sysCfg.GET("/key/:key", middleware.RequirePermission(permRepo, "config:system"), sysConfigHandler.GetByKey)
				sysCfg.GET("/:id", middleware.RequirePermission(permRepo, "config:system"), sysConfigHandler.Get)
				sysCfg.POST("", middleware.RequirePermission(permRepo, "config:system"), sysConfigHandler.Create)
				sysCfg.PUT("/:id", middleware.RequirePermission(permRepo, "config:system"), sysConfigHandler.Update)
				sysCfg.DELETE("/:id", middleware.RequirePermission(permRepo, "config:system"), sysConfigHandler.Delete)
			}

			// 公告管理（路由顺序：静态段 /export、/:id/publish|revoke 优先于 /:id 注册）
			notice := sys.Group("/notice")
			{
				notice.GET("/export", middleware.RequirePermission(permRepo, "notice:list"), noticeHandler.Export)
				notice.GET("", middleware.RequirePermission(permRepo, "notice:list"), noticeHandler.List)
				notice.POST("/:id/publish", middleware.RequirePermission(permRepo, "notice:edit"), noticeHandler.Publish)
				notice.POST("/:id/revoke", middleware.RequirePermission(permRepo, "notice:edit"), noticeHandler.Revoke)
				notice.GET("/:id", middleware.RequirePermission(permRepo, "notice:list"), noticeHandler.Get)
				notice.POST("", middleware.RequirePermission(permRepo, "notice:create"), noticeHandler.Create)
				notice.PUT("/:id", middleware.RequirePermission(permRepo, "notice:edit"), noticeHandler.Update)
				notice.DELETE("/:id", middleware.RequirePermission(permRepo, "notice:delete"), noticeHandler.Delete)
				notice.DELETE("", middleware.RequirePermission(permRepo, "notice:delete"), noticeHandler.BatchDelete)
			}
		}
	}

	return r
}
