package dbmysql

import (
	"time"
)

type Device struct {
	DeviceToken  string    `gorm:"primaryKey;size:255"`
	UserID       string    `gorm:"not null;index;size:36`
	Platform     string    `gorm:"not null;size:10"`
	RegisteredAt time.Time `gorm:"autoCreateTime"`
	LastActive   time.Time `gorm:"autoCreateTime"`
}

func (Device) TableName() string {
	return "devices"
}
