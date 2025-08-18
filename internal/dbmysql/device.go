package dbmysql

import (
	"time"
)

type Device struct {
	DeviceToken string `gorm:"primaryKey;column:device_token;size:255" json:"device_token"`
	// UserID       string    `gorm:"column:user_id;not null;index" json:"user_id"`

	// UserID       uint64    `gorm:"primaryKey;column:user_id;autoIncrement;not null" json:"user_id"`
	// Platform     string    `gorm:"not null;size:10""column:platform;type:enum('android','ios','web')" json:"platform"`
	UserID       uint64    `gorm:"column:user_id;not null;index" json:"user_id"`
	Platform     string    `gorm:"not null;size:10;column:platform;type:enum('android','ios','web')" json:"platform"`
	RegisteredAt time.Time `gorm:"column:registered_at;autoCreateTime" json:"registered_at"`
	LastActive   time.Time `gorm:"column:last_active;autoUpdateTime" json:"last_active"`
	User         User      `gorm:"-" json:"user,omitempty"`


}
