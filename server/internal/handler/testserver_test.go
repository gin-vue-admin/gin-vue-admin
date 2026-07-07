package handler

import (
	"context"
	"net/http"
	"testing"

	"gva/internal/config"
	"gva/internal/middleware"
	"gva/internal/pkg/async"
	"gva/internal/pkg/audit"
	"gva/internal/pkg/datascope"
	"gva/internal/pkg/jwt"
	"gva/internal/repository"
	"gva/internal/service"
	"gva/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// newAppServer 装配全量模块的最小 gin 引擎（复用 main.go 依赖组装 + router.go 路由结构），
// 用 SQLite + SyncRunner 隔离跑通全链路。返回 engine 与 admin token，供各模块集成测试共享。
//
// 路由含 AuthRequired + RequirePermission；admin 种子持 "*" 超管权限，短路放行所有端点。
// 不挂 OperationLog 中间件（写操作日志记录由 middleware 包自身测试覆盖），聚焦 handler 逻辑。
func newAppServer(t *testing.T) (r *gin.Engine, adminToken string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutil.NewTestDB(t)
	require.NoError(t, audit.Register(db))

	jwtMgr := jwt.NewManager(config.JWTConfig{
		Secret: "test-secret-very-long-for-hs256-signing", AccessTTL: 3600, RefreshTTL: 86400, Issuer: "gva-test",
	})
	runner := async.SyncRunner{}

	// auth + 种子（admin/user 两用户，super_admin 角色持 "*" 权限）
	userRepo := repository.NewUserRepository(db)
	loginLogRepo := repository.NewLoginLogRepository(db)
	authSvc := service.NewAuthService(userRepo, db, jwtMgr, runner, loginLogRepo)
	require.NoError(t, authSvc.Seed(context.Background()))

	// 权限 / 角色 / 部门
	permRepo := repository.NewPermissionRepository(db)
	permSvc := service.NewPermissionService(permRepo)
	roleRepo := repository.NewRoleRepository(db)
	roleSvc := service.NewRoleService(roleRepo)
	deptRepo := repository.NewDeptRepository(db)

	// 用户（数据范围 Resolver）
	userSvc := service.NewUserService(userRepo, datascope.NewResolver(userRepo, deptRepo))

	// 菜单 / crud / 部门
	menuSvc := service.NewMenuService(repository.NewMenuRepository(db))
	crudSvc := service.NewCrudService(repository.NewCrudRepository(db))
	deptSvc := service.NewDeptService(deptRepo)

	// 字典三级
	dictCatSvc := service.NewDictCategoryService(repository.NewDictCategoryRepository(db))
	dictSvc := service.NewDictService(repository.NewDictRepository(db))
	dictItemSvc := service.NewDictItemService(repository.NewDictItemRepository(db))

	// 日志
	opLogRepo := repository.NewOperationLogRepository(db)
	opLogSvc := service.NewOperationLogService(opLogRepo)
	loginLogSvc := service.NewLoginLogService(loginLogRepo)

	// 系统配置（预热内存缓存）
	sysConfigSvc := service.NewSysConfigService(repository.NewSysConfigRepository(db))
	sysConfigSvc.SetJWTManager(jwtMgr)
	require.NoError(t, sysConfigSvc.LoadAll(context.Background()))

	// dashboard / notice
	dashboardSvc := service.NewDashboardService(db)
	noticeSvc := service.NewNoticeService(repository.NewNoticeRepository(db), userRepo)

	// handler 装配
	authH := NewAuthHandler(authSvc)
	permH := NewPermissionHandler(permSvc)
	roleH := NewRoleHandler(roleSvc)
	userH := NewUserHandler(userSvc)
	menuH := NewMenuHandler(menuSvc)
	crudH := NewCrudHandler(crudSvc)
	deptH := NewDeptHandler(deptSvc)
	dictCatH := NewDictCategoryHandler(dictCatSvc)
	dictH := NewDictHandler(dictSvc)
	dictItemH := NewDictItemHandler(dictItemSvc)
	opLogH := NewOperationLogHandler(opLogSvc)
	loginLogH := NewLoginLogHandler(loginLogSvc)
	sysConfigH := NewSysConfigHandler(sysConfigSvc)
	dashboardH := NewDashboardHandler(dashboardSvc)
	noticeH := NewNoticeHandler(noticeSvc)
	healthH := NewHealthHandler()

	r = gin.New()
	r.HandleMethodNotAllowed = true
	api := r.Group("/api")
	{
		api.GET("/health", healthH.Health)
		api.GET("/system/config/public", sysConfigH.GetPublic)

		dashboard := api.Group("/dashboard", middleware.AuthRequired(jwtMgr))
		{
			dashboard.GET("/stats", dashboardH.Stats)
			dashboard.GET("/charts", dashboardH.Charts)
			dashboard.GET("/activities", dashboardH.Activities)
		}

		auth := api.Group("/auth")
		{
			auth.POST("/sessions", authH.Login)
			auth.DELETE("/sessions", authH.Logout)
			auth.POST("/tokens/refresh", authH.Refresh)
			auth.GET("/users/me", middleware.AuthRequired(jwtMgr), authH.Me)
		}

		perm := api.Group("/permission", middleware.AuthRequired(jwtMgr))
		{
			perm.GET("/export", middleware.RequirePermission(permRepo, "permission:list"), permH.Export)
			perm.GET("", middleware.RequirePermission(permRepo, "permission:list"), permH.List)
			perm.GET("/:id", middleware.RequirePermission(permRepo, "permission:list"), permH.Get)
			perm.POST("", middleware.RequirePermission(permRepo, "permission:create"), permH.Create)
			perm.PUT("/:id", middleware.RequirePermission(permRepo, "permission:edit"), permH.Update)
			perm.DELETE("/:id", middleware.RequirePermission(permRepo, "permission:delete"), permH.Delete)
			perm.DELETE("", middleware.RequirePermission(permRepo, "permission:delete"), permH.BatchDelete)
		}

		role := api.Group("/role", middleware.AuthRequired(jwtMgr))
		{
			role.GET("/export", middleware.RequirePermission(permRepo, "role:list"), roleH.Export)
			role.GET("/:id/permissions", middleware.RequirePermission(permRepo, "role:permission"), roleH.GetPermissions)
			role.PUT("/:id/permissions", middleware.RequirePermission(permRepo, "role:permission"), roleH.SetPermissions)
			role.GET("", middleware.RequirePermission(permRepo, "role:list"), roleH.List)
			role.GET("/:id", middleware.RequirePermission(permRepo, "role:list"), roleH.Get)
			role.POST("", middleware.RequirePermission(permRepo, "role:create"), roleH.Create)
			role.PUT("/:id", middleware.RequirePermission(permRepo, "role:edit"), roleH.Update)
			role.DELETE("/:id", middleware.RequirePermission(permRepo, "role:delete"), roleH.Delete)
			role.DELETE("", middleware.RequirePermission(permRepo, "role:delete"), roleH.BatchDelete)
		}

		user := api.Group("/user", middleware.AuthRequired(jwtMgr))
		{
			user.GET("/export", middleware.RequirePermission(permRepo, "user:list"), userH.Export)
			user.GET("", middleware.RequirePermission(permRepo, "user:list"), userH.List)
			user.GET("/:id", middleware.RequirePermission(permRepo, "user:list"), userH.Get)
			user.POST("", middleware.RequirePermission(permRepo, "user:create"), userH.Create)
			user.PUT("/:id", middleware.RequirePermission(permRepo, "user:edit"), userH.Update)
			user.DELETE("/:id", middleware.RequirePermission(permRepo, "user:delete"), userH.Delete)
			user.DELETE("", middleware.RequirePermission(permRepo, "user:delete"), userH.BatchDelete)
		}

		crud := api.Group("/crud", middleware.AuthRequired(jwtMgr))
		{
			crud.GET("", middleware.RequirePermission(permRepo, "crud:list"), crudH.List)
			crud.GET("/:id", middleware.RequirePermission(permRepo, "crud:list"), crudH.Get)
			crud.POST("", middleware.RequirePermission(permRepo, "crud:create"), crudH.Create)
			crud.PUT("/:id", middleware.RequirePermission(permRepo, "crud:edit"), crudH.Update)
			crud.DELETE("/:id", middleware.RequirePermission(permRepo, "crud:delete"), crudH.Delete)
			crud.DELETE("", middleware.RequirePermission(permRepo, "crud:delete"), crudH.BatchDelete)
		}

		dict := api.Group("/dict", middleware.AuthRequired(jwtMgr))
		registerTestCRUDRoutes(dict.Group("/categories"), permRepo, "dict", dictCatH)
		registerTestCRUDRoutes(dict.Group("/dicts"), permRepo, "dict", dictH)
		registerTestCRUDRoutes(dict.Group("/items"), permRepo, "dict", dictItemH)

		sys := api.Group("/system", middleware.AuthRequired(jwtMgr))
		{
			sys.GET("/menus", menuH.Menus)

			menu := sys.Group("/menu")
			{
				menu.GET("", middleware.RequirePermission(permRepo, "menu:list"), menuH.GetTree)
				menu.POST("", middleware.RequirePermission(permRepo, "menu:create"), menuH.Create)
				menu.PUT("/:id", middleware.RequirePermission(permRepo, "menu:edit"), menuH.Update)
				menu.DELETE("/:id", middleware.RequirePermission(permRepo, "menu:delete"), menuH.Delete)
				menu.PATCH("/sort", middleware.RequirePermission(permRepo, "menu:edit"), menuH.Sort)
			}

			dept := sys.Group("/dept")
			{
				dept.GET("/export", middleware.RequirePermission(permRepo, "dept:list"), deptH.Export)
				dept.GET("", middleware.RequirePermission(permRepo, "dept:list"), deptH.List)
				dept.GET("/:id", middleware.RequirePermission(permRepo, "dept:list"), deptH.Get)
				dept.POST("", middleware.RequirePermission(permRepo, "dept:create"), deptH.Create)
				dept.PUT("/:id", middleware.RequirePermission(permRepo, "dept:edit"), deptH.Update)
				dept.DELETE("/:id", middleware.RequirePermission(permRepo, "dept:delete"), deptH.Delete)
				dept.DELETE("", middleware.RequirePermission(permRepo, "dept:delete"), deptH.BatchDelete)
			}

			opLog := sys.Group("/operation-log")
			{
				opLog.GET("", middleware.RequirePermission(permRepo, "system:log"), opLogH.List)
				opLog.DELETE("/clear", middleware.RequirePermission(permRepo, "system:log"), opLogH.Clear)
				opLog.DELETE("", middleware.RequirePermission(permRepo, "system:log"), opLogH.BatchDelete)
				opLog.GET("/:id", middleware.RequirePermission(permRepo, "system:log"), opLogH.Get)
				opLog.DELETE("/:id", middleware.RequirePermission(permRepo, "system:log"), opLogH.Delete)
			}

			loginLog := sys.Group("/login-log")
			{
				loginLog.GET("", middleware.RequirePermission(permRepo, "system:log"), loginLogH.List)
				loginLog.DELETE("/clear", middleware.RequirePermission(permRepo, "system:log"), loginLogH.Clear)
				loginLog.DELETE("", middleware.RequirePermission(permRepo, "system:log"), loginLogH.BatchDelete)
				loginLog.GET("/:id", middleware.RequirePermission(permRepo, "system:log"), loginLogH.Get)
				loginLog.DELETE("/:id", middleware.RequirePermission(permRepo, "system:log"), loginLogH.Delete)
			}

			sysCfg := sys.Group("/config")
			{
				sysCfg.GET("", middleware.RequirePermission(permRepo, "config:system"), sysConfigH.List)
				sysCfg.GET("/key/:key", middleware.RequirePermission(permRepo, "config:system"), sysConfigH.GetByKey)
				sysCfg.GET("/:id", middleware.RequirePermission(permRepo, "config:system"), sysConfigH.Get)
				sysCfg.POST("", middleware.RequirePermission(permRepo, "config:system"), sysConfigH.Create)
				sysCfg.PUT("/:id", middleware.RequirePermission(permRepo, "config:system"), sysConfigH.Update)
				sysCfg.DELETE("/:id", middleware.RequirePermission(permRepo, "config:system"), sysConfigH.Delete)
			}

			notice := sys.Group("/notice")
			{
				notice.GET("/export", middleware.RequirePermission(permRepo, "notice:list"), noticeH.Export)
				notice.GET("", middleware.RequirePermission(permRepo, "notice:list"), noticeH.List)
				notice.POST("/:id/publish", middleware.RequirePermission(permRepo, "notice:edit"), noticeH.Publish)
				notice.POST("/:id/revoke", middleware.RequirePermission(permRepo, "notice:edit"), noticeH.Revoke)
				notice.GET("/:id", middleware.RequirePermission(permRepo, "notice:list"), noticeH.Get)
				notice.POST("", middleware.RequirePermission(permRepo, "notice:create"), noticeH.Create)
				notice.PUT("/:id", middleware.RequirePermission(permRepo, "notice:edit"), noticeH.Update)
				notice.DELETE("/:id", middleware.RequirePermission(permRepo, "notice:delete"), noticeH.Delete)
				notice.DELETE("", middleware.RequirePermission(permRepo, "notice:delete"), noticeH.BatchDelete)
			}
		}
	}

	// admin 登录拿 token
	w := doJSON(t, r, http.MethodPost, "/api/auth/sessions",
		LoginRequest{Username: "admin", Password: "123456"})
	require.Equal(t, 200, w.Code, w.Body.String())
	data, _ := decodeResult(t, w).Data.(map[string]any)
	return r, data["accessToken"].(string)
}

// testCRUDHandler 字典三级 handler 共同满足的接口，用于 registerTestCRUDRoutes 消除重复注册。
// 与 server.registerDictRoutes 同构，仅在 handler 包内复刻以避免循环导入。
type testCRUDHandler interface {
	Export(c *gin.Context)
	List(c *gin.Context)
	Get(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
	BatchDelete(c *gin.Context)
}

// registerTestCRUDRoutes 注册标准 CRUD 七端点（export/list/get/create/update/delete/batchDelete）。
func registerTestCRUDRoutes(g *gin.RouterGroup, permRepo repository.PermissionRepository, code string, h testCRUDHandler) {
	g.GET("/export", middleware.RequirePermission(permRepo, code+":list"), h.Export)
	g.GET("", middleware.RequirePermission(permRepo, code+":list"), h.List)
	g.GET("/:id", middleware.RequirePermission(permRepo, code+":list"), h.Get)
	g.POST("", middleware.RequirePermission(permRepo, code+":create"), h.Create)
	g.PUT("/:id", middleware.RequirePermission(permRepo, code+":edit"), h.Update)
	g.DELETE("/:id", middleware.RequirePermission(permRepo, code+":delete"), h.Delete)
	g.DELETE("", middleware.RequirePermission(permRepo, code+":delete"), h.BatchDelete)
}
