// Package db 负责数据库连接初始化。当前为 MySQL(GORM)，结构上便于后续扩展多驱动。
package db

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gva/internal/config"
)

// NewMySQL 建立 MySQL 连接并配置连接池参数。
func NewMySQL(cfg config.DBConfig, mode string) (*gorm.DB, error) {
	logLevel := logger.Warn
	if mode == "debug" {
		logLevel = logger.Info
	}
	gdb, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("连接 MySQL 失败: %w", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层 SQL DB 失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConn)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConn)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return gdb, nil
}
