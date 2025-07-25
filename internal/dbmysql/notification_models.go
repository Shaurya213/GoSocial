package dbmysql

import (
	"gosocial/internal/common"
	"time"
)

type Notification struct { //Notification struct contaning all the attributes for notification table
	ID            string  `gorm:"primaryKey;size:36"`
	UserID        string  `gorm:"not null;index;size:36"`
	Header        string  `gorm:"not null;size:255"`
	Content       string  `gorm:"not null;type:text"`
	ImageURL      *string `gorm:"size:512"`
	ScheduledAt   *time.Time
	SentAt        *time.Time
	ReadAt        *time.Time
	Type          string                      `gorm:"not null;size:50"`
	Status        string                      `gorm:"default:'pending';size:50"`
	Priority      int                         `gorm:"default:1"`
	TriggerUserID *string                     `gorm:"size:36"`
	ContentID     *string                     `gorm:"size:36"`
	Metadata      common.NotificationMetadata `gorm:"type:json"`
	RetryCount    int                         `gorm:"default:0"`
	CreatedAt     time.Time                   `gorm:"autoCreateTime"`
	UpdatedAt     time.Time                   `gorm:"autoUpdateTime"`
}

type Device struct {
	DeviceToken  string    `gorm:"primaryKey;size:255"`
	UserID       string    `gorm:"not null;index;size:36"`
	Platform     string    `gorm:"not null;size:10"`
	RegisteredAt time.Time `gorm:"autoCreateTime"`
	LastActive   time.Time `gorm:"autoCreateTime"`
}
