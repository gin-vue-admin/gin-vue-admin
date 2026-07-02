// Package log 提供全局 zap 日志器，支持 console（彩色）与 file（JSON + lumberjack 滚动）两种模式。
package log

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gva/internal/config"
)

var (
	L *zap.Logger
	S *zap.SugaredLogger
)

// init 提供 console 默认日志器，确保 config.Init 之前也可安全使用。
func init() {
	L = zap.NewNop()
	S = L.Sugar()
}

var initOnce sync.Once

// Init 按 LogConfig 初始化全局日志器（仅生效一次，后续调用被忽略）。
func Init(cfg config.LogConfig) {
	initOnce.Do(func() {
		var level zapcore.Level
		if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
			level = zapcore.InfoLevel
		}

		var ws zapcore.WriteSyncer
		if cfg.Mode == "file" {
			ws = zapcore.AddSync(&lumberjack.Logger{
				Filename:   cfg.Filename,
				MaxSize:    cfg.MaxSize,
				MaxAge:     cfg.MaxAge,
				MaxBackups: cfg.MaxBackups,
				Compress:   true,
			})
		} else {
			ws = zapcore.Lock(os.Stdout)
		}

		encCfg := zap.NewProductionEncoderConfig()
		encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		encCfg.EncodeLevel = zapcore.CapitalLevelEncoder
		var enc zapcore.Encoder = zapcore.NewConsoleEncoder(encCfg)
		if cfg.Mode == "file" {
			enc = zapcore.NewJSONEncoder(encCfg)
		}

		L = zap.New(zapcore.NewCore(enc, ws, level), zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
		S = L.Sugar()
	})
}
