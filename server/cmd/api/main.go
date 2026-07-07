// @title           GVA 通用管理框架基座 API
// @version         1.0
// @description     业界优秀的通用管理框架基座（非业务系统）：RBAC 权限、菜单、部门、字典、操作日志、登录日志、数据范围、系统配置。
// @description     清洁架构 handler → service → repository → model，复用泛型 Repository 基类。
// @host            localhost:8088
// @BasePath        /api
// @schemes         http https
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     "Bearer {accessToken}"
package main

import (
	"context"
	"flag"
	"fmt"
	"strconv"

	_ "gva/docs" // M8 Swagger：触发 docs 包 init，注册 swagger spec
	"gva/internal/config"
	"gva/internal/db"
	"gva/internal/handler"
	"gva/internal/model"
	"gva/internal/pkg/async"
	"gva/internal/pkg/audit"
	"gva/internal/pkg/datascope"
	"gva/internal/pkg/jwt"
	"gva/internal/pkg/log"
	"gva/internal/repository"
	"gva/internal/server"
	"gva/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// version 版本号，可由构建时 ldflags 注入：-ldflags "-X main.version=0.1.0"
var version = "0.1.0"

func main() {
	// -version：打印版本号并退出（开源项目常备）
	showVersion := flag.Bool("version", false, "打印版本号并退出")
	flag.Parse()
	if *showVersion {
		fmt.Println("gva version", version)
		return
	}

	// 1. 配置
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// 2. 日志
	log.Init(cfg.Log)
	gin.SetMode(cfg.App.Mode)

	// 3. 数据库 + 审计回调 + 自动迁移
	gdb, err := db.NewMySQL(cfg.DB, cfg.App.Mode)
	if err != nil {
		log.L.Fatal("初始化数据库失败", zap.Error(err))
	}
	if err := audit.Register(gdb); err != nil { // 注册 GORM 审计回调：Create/Update 自动写入 CreatedBy/UpdatedBy
		log.L.Fatal("注册审计回调失败", zap.Error(err))
	}
	if err := model.AutoMigrate(gdb); err != nil {
		log.L.Fatal("自动迁移失败", zap.Error(err))
	}
	log.L.Info("数据库初始化与迁移完成")

	// 4. 组装依赖（构造注入）
	jwtMgr := jwt.NewManager(cfg.JWT)
	runner := async.GoroutineRunner{}
	userRepo := repository.NewUserRepository(gdb)
	loginLogRepo := repository.NewLoginLogRepository(gdb) // M8 登录日志：AuthService 与 LoginLogHandler 共用
	authSvc := service.NewAuthService(userRepo, gdb, jwtMgr, runner, loginLogRepo)
	if err := authSvc.Seed(context.Background()); err != nil {
		log.L.Fatal("种子数据初始化失败", zap.Error(err))
	}
	log.L.Info("种子数据就绪")

	// M3.1: 权限模块组装
	permRepo := repository.NewPermissionRepository(gdb)
	permSvc := service.NewPermissionService(permRepo)
	permHandler := handler.NewPermissionHandler(permSvc)

	// M3.2: 角色模块组装（repo → service → handler 构造注入）
	roleRepo := repository.NewRoleRepository(gdb)
	roleSvc := service.NewRoleService(roleRepo)
	roleHandler := handler.NewRoleHandler(roleSvc)

	// M3.3: 用户模块组装（userRepo 复用 M2 已建，再 service → handler）
	// M8 数据范围：Resolver 注入 UserService，按当前用户角色过滤可见用户。
	// deptRepo 提前创建以供 Resolver 复用（M7 部门块不再重复创建）。
	deptRepo := repository.NewDeptRepository(gdb)
	userSvc := service.NewUserService(userRepo, datascope.NewResolver(userRepo, deptRepo))
	userHandler := handler.NewUserHandler(userSvc)

	// M4.1: 菜单模块组装
	menuRepo := repository.NewMenuRepository(gdb)
	menuSvc := service.NewMenuService(menuRepo)
	menuHandler := handler.NewMenuHandler(menuSvc)

	// M6: 通用 CRUD 示例（脚手架范例，复用泛型 Repository 基类）
	crudRepo := repository.NewCrudRepository(gdb)
	crudSvc := service.NewCrudService(crudRepo)
	crudHandler := handler.NewCrudHandler(crudSvc)

	// M7: 部门管理（树形 + 级联删除）；deptRepo 已在用户模块处提前创建供 Resolver 复用。
	deptSvc := service.NewDeptService(deptRepo)
	deptHandler := handler.NewDeptHandler(deptSvc)

	// M7: 字典管理（三级：分类/字典/字典项）
	dictCatRepo := repository.NewDictCategoryRepository(gdb)
	dictRepo := repository.NewDictRepository(gdb)
	dictItemRepo := repository.NewDictItemRepository(gdb)
	dictCatHandler := handler.NewDictCategoryHandler(service.NewDictCategoryService(dictCatRepo))
	dictHandler := handler.NewDictHandler(service.NewDictService(dictRepo))
	dictItemHandler := handler.NewDictItemHandler(service.NewDictItemService(dictItemRepo))

	// M8: 操作日志（中间件异步记录写操作）
	opLogRepo := repository.NewOperationLogRepository(gdb)
	opLogHandler := handler.NewOperationLogHandler(service.NewOperationLogService(opLogRepo))

	// M8: 登录日志（repo 已在 authSvc 装配时创建，此处仅组装查询端 handler）
	loginLogHandler := handler.NewLoginLogHandler(service.NewLoginLogService(loginLogRepo))

	// M10: 系统参数配置（seed 已种入内置 key，此处预热内存缓存）
	sysConfigRepo := repository.NewSysConfigRepository(gdb)
	sysConfigSvc := service.NewSysConfigService(sysConfigRepo)
	// sys_config 接入：token_expire_seconds 实时同步到 jwtMgr（applyRuntime 在 LoadAll/reload 后调用）
	sysConfigSvc.SetJWTManager(jwtMgr)
	if err := sysConfigSvc.LoadAll(context.Background()); err != nil {
		log.L.Fatal("系统配置缓存加载失败", zap.Error(err))
	}
	sysConfigHandler := handler.NewSysConfigHandler(sysConfigSvc)

	// 首页统计（dashboard 聚合：用户/角色/权限/菜单计数 + 操作日志活动流与趋势）
	dashboardHandler := handler.NewDashboardHandler(service.NewDashboardService(gdb))

	// 公告管理（notice CRUD + 发布/撤销 + 导出；userRepo 用于反查发布人用户名）
	noticeRepo := repository.NewNoticeRepository(gdb)
	noticeHandler := handler.NewNoticeHandler(service.NewNoticeService(noticeRepo, userRepo))

	authHandler := handler.NewAuthHandler(authSvc)
	r := server.NewRouter(authHandler, permHandler, roleHandler, userHandler, menuHandler, crudHandler, deptHandler, dictCatHandler, dictHandler, dictItemHandler, opLogHandler, loginLogHandler, sysConfigHandler, dashboardHandler, noticeHandler, permRepo, opLogRepo, jwtMgr, runner)

	// 5. 启动 HTTP 服务
	addr := ":" + strconv.Itoa(cfg.Server.Port)
	log.L.Info("gva server listening", zap.String("addr", addr))
	if err := r.Run(addr); err != nil {
		log.L.Fatal("server failed", zap.Error(err))
	}
}
