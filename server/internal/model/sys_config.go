package model

// SysConfig 系统参数配置（key-value）。运营人员在后台改值即生效（无需发版），
// 是基座区别于"系统"的成熟度标志（对标 RuoYi sys_config）。
//
// 内置常用 key 由 service.AuthService.Seed 初始化：site_title / default_page_size /
// login_captcha_enabled / token_expire_seconds 等。Type 标注值的解析方式（string/bool/int/json）。
//
// 编程取值走 service.SysConfigService.Get / GetBool / GetInt（内存缓存），不直接打库。
type SysConfig struct {
	Model
	ConfigKey   string `gorm:"size:128;uniqueIndex;not null" json:"configKey"` // 全局唯一键
	ConfigValue string `gorm:"type:text" json:"configValue"`                   // 值（统一字符串存储，按 Type 解析）
	ConfigName  string `gorm:"size:128" json:"configName"`                     // 展示名
	Remark      string `gorm:"size:255" json:"remark"`                         // 说明
	Type        string `gorm:"size:32;default:string" json:"type"`             // string/bool/int/json
}

func (SysConfig) TableName() string { return "sys_config" }
