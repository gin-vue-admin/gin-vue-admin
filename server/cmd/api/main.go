package main

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gva/internal/config"
	"gva/internal/db"
	"gva/internal/handler"
	"gva/internal/model"
	"gva/internal/pkg/async"
	"gva/internal/pkg/jwt"
	"gva/internal/pkg/log"
	"gva/internal/repository"
	"gva/internal/server"
	"gva/internal/service"
)

func main() {
	// 1. 配置
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// 2. 日志
	log.Init(cfg.Log)
	gin.SetMode(cfg.App.Mode)

	// 3. 数据库 + 自动迁移
	gdb, err := db.NewMySQL(cfg.DB, cfg.App.Mode)
	if err != nil {
		log.L.Fatal("初始化数据库失败", zap.Error(err))
	}
	if err := model.AutoMigrate(gdb); err != nil {
		log.L.Fatal("自动迁移失败", zap.Error(err))
	}
	log.L.Info("数据库初始化与迁移完成")

	// 4. 组装依赖（构造注入）
	jwtMgr := jwt.NewManager(cfg.JWT)
	userRepo := repository.NewUserRepository(gdb)
	authSvc := service.NewAuthService(userRepo, gdb, jwtMgr, async.GoroutineRunner{})
	if err := authSvc.Seed(context.Background()); err != nil {
		log.L.Fatal("种子数据初始化失败", zap.Error(err))
	}
	log.L.Info("种子数据就绪")

	authHandler := handler.NewAuthHandler(authSvc)
	r := server.NewRouter(authHandler, jwtMgr)

	// 5. 启动 HTTP 服务
	addr := ":" + strconv.Itoa(cfg.Server.Port)
	log.L.Info("gva server listening", zap.String("addr", addr))
	if err := r.Run(addr); err != nil {
		log.L.Fatal("server failed", zap.Error(err))
	}
}
