// Package config 集中管理应用配置：configs/config.yaml + GVA_ 前缀环境变量覆盖。
// 环境变量用双下划线分隔嵌套，如 GVA_DB__PASSWORD 覆盖 db.password。
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config 顶层配置聚合。
type Config struct {
	App    AppConfig    `mapstructure:"app"`
	Server ServerConfig `mapstructure:"server"`
	DB     DBConfig     `mapstructure:"db"`
	JWT    JWTConfig    `mapstructure:"jwt"`
	Log    LogConfig    `mapstructure:"log"`
}

// AppConfig 应用级配置（名称/运行模式）。
type AppConfig struct {
	Name string `mapstructure:"name"`
	Mode string `mapstructure:"mode"` // debug | release
}

// ServerConfig HTTP 服务配置。
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// DBConfig 数据库（MySQL）连接配置。
type DBConfig struct {
	Driver      string `mapstructure:"driver"`
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	User        string `mapstructure:"user"`
	Password    string `mapstructure:"password"`
	DBName      string `mapstructure:"dbname"`
	Charset     string `mapstructure:"charset"`
	MaxOpenConn int    `mapstructure:"maxOpenConn"`
	MaxIdleConn int    `mapstructure:"maxIdleConn"`
}

// DSN 生成 MySQL 连接串。
func (d DBConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.Charset)
}

// JWTConfig 令牌签发配置（密钥/TTL/签发方）。
type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	AccessTTL  int    `mapstructure:"accessTTL"`  // 秒
	RefreshTTL int    `mapstructure:"refreshTTL"` // 秒
	Issuer     string `mapstructure:"issuer"`
}

// LogConfig 日志配置（级别/输出模式/滚动参数）。
type LogConfig struct {
	Level      string `mapstructure:"level"` // debug | info | warn | error
	Mode       string `mapstructure:"mode"`  // console | file
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"maxSize"` // MB
	MaxAge     int    `mapstructure:"maxAge"`  // 天
	MaxBackups int    `mapstructure:"maxBackups"`
}

// Load 依次从 configs/ 目录加载 config.yaml，并启用环境变量覆盖。
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("configs")
	v.AddConfigPath("./configs")
	v.AddConfigPath("/app/configs")

	v.SetEnvPrefix("GVA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	v.AutomaticEnv()

	// 默认值
	v.SetDefault("app.mode", "debug")
	v.SetDefault("server.port", 8080)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}
	return &c, nil
}
