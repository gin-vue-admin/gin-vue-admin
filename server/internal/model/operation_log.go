package model

// OperationLog 操作日志（由 middleware.OperationLog 自动记录写操作 POST/PUT/DELETE）。
// 嵌入 Model 获审计字段（CreatedBy 记录操作者）与软删；Clear 用硬删清空。
type OperationLog struct {
	Model
	Username  string `gorm:"size:64;index" json:"username"`
	Method    string `gorm:"size:16" json:"method"`      // GET/POST/PUT/DELETE
	Path      string `gorm:"size:255;index" json:"path"` // 请求路径
	Params    string `gorm:"type:text" json:"params"`    // 请求体（JSON 字符串）
	IP        string `gorm:"size:64" json:"ip"`
	UserAgent string `gorm:"size:255" json:"userAgent"`
	Status    string `gorm:"size:16" json:"status"` // success | failed
	HTTPCode  int    `json:"httpCode"`
	Duration  int64  `json:"duration"` // 耗时（毫秒）
}

// TableName 表名复数。
func (OperationLog) TableName() string { return "operation_logs" }
