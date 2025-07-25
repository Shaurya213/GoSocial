package dbmysql

import "time"

type Notification struct {
	NotificationID uint64    `gorm:"primaryKey;autoIncrement" json:"notification_id"`
	UserID         uint64    `gorm:"index;not null" json:"user_id"`
	Content        string    `gorm:"type:text" json:"content"`
	Type           string    `gorm:"type:enum('system','friend','reaction','chat'); not null" json:"type"`
	Status         string    `gorm:"type:enum('sent','delivered','read');default:'sent'" json:"status"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	ReadAt         time.Time `json:"read_at"`
}
