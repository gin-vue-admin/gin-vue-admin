package model

import "time"

// Notice 系统公告（对标 RuoYi sys_notice）。
type Notice struct {
	Model
	Title       string     `gorm:"size:255;not null" json:"title"`
	Content     string     `gorm:"type:text" json:"content"`
	Type        string     `gorm:"size:16;index;default:notice" json:"type"`  // announcement | notice | todo
	Status      string     `gorm:"size:16;index;default:draft" json:"status"` // published | draft | expired
	Priority    string     `gorm:"size:16;default:medium" json:"priority"`    // high | medium | low
	PublishTime *time.Time `gorm:"index" json:"publishTime,omitempty"`
	ExpireTime  *time.Time `gorm:"index" json:"expireTime,omitempty"`
	Publisher   string     `gorm:"size:64" json:"publisher"`
}

// TableName 表名复数。
func (Notice) TableName() string { return "notices" }
