package model

// LoginLog 登录日志（由 AuthService.Login 在每次登录尝试后异步记录，无论成功失败）。
// 嵌入 Model：CreatedAt 即登录时间；Clear 用硬删清空。
// 失败原因 Reason 仅供内部审计，与 HTTP 响应文案解耦（响应统一"用户名或密码错误"防枚举）。
type LoginLog struct {
	Model
	Username  string `gorm:"size:64;index" json:"username"` // 登录尝试的用户名（成功失败均记）
	Status    string `gorm:"size:16;index" json:"status"`   // success | failed
	IP        string `gorm:"size:64;index" json:"ip"`       // 客户端 IP
	UserAgent string `gorm:"size:255" json:"userAgent"`     // User-Agent
	Reason    string `gorm:"size:255" json:"reason"`        // 失败原因（成功为空）
}

// TableName 表名复数。
func (LoginLog) TableName() string { return "login_logs" }
