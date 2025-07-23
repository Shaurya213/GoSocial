package dbmysql

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type NotificationsType string

const (
	FriendRequestType NotificationsType = "friend_request"
	PostReactionType  NotificationsType = "post_reaction"
	MessageType       NotificationsType = "message"
	StoryReactionType NotificationsType = "story_reaction"
	SystemType        NotificationsType = "system"
)

type NotificationStatus string

const (
	StatusPending   NotificationStatus = "pending"
	StatusScheduled NotificationStatus = "scheduled"
	StatusSent      NotificationStatus = "sent"
	StatusDelivered NotificationStatus = "delivered"
	StatusFailed    NotificationStatus = "failed"
	StatusRead      NotificationStatus = "read"
)

type NotificationMetadata map[string]interface{}

func (nm *NotificationMetadata) Scan(value interface{}) error {
	if value == nil {
		*nm = make(NotificationMetadata)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into NotificationMetadata", value)
	}
	return json.Unmarshal(bytes, nm)
}

func (nm NotificationMetadata) Value() (driver.Value, error) {
	return json.Marshal(nm)
}

type Notification struct {
	ID             string  `gorm:"primaryKey;size:36"`
	UserID         string  `gorm:"not null;index;size:36"`
	Header         string  `gorm:"not null;size:255"`
	Content        string  `gorm:"not null; type:text"`
	ImageURL       *string `gorm:"size:512"`
	ScheduledAt    *time.Time
	SentAt         *time.Time
	ReadAt         *time.Time
	Type           NotificationsType    `gorm:"not null; size:50"`
	Status         NotificationStatus   `gorm:"default:'pending';size:50`
	Priority       int                  `gorm:"default:1"`
	Metadata       NotificationMetadata `gorm:"size:36"`
	RetryCount     int                  `gorm:"size:36"`
	CreatedAt      time.Time            `gorm:"type:json"`
	UpdatedAt      time.Time            `gorm:"default:0"`
	TrigegerUserID *string              `gorm:"autoCreateTime"`
	ContentID      *string              `gorm:"autoUpdateTime"`
}

func (Notification) TableName() string {
	return "notifications"
}
