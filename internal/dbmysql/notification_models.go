package dbmysql

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"gosocial/internal/common"
	"time"
)

type Notification struct {
	ID            string  `gorm:"primaryKey;size:36"`
	UserID        string  `gorm:"not null;index;size:36"`
	Header        string  `gorm:"not null;size:255"`
	Content       string  `gorm:"not null;type:text"`
	ImageURL      *string `gorm:"size:512"`
	ScheduledAt   *time.Time
	SentAt        *time.Time
	ReadAt        *time.Time
	Type          common.NotificationType   `gorm:"not null;size:50"`
	Status        common.NotificationStatus `gorm:"default:'pending';size:50"`
	Priority      int                       `gorm:"default:1"`
	TriggerUserID *string                   `gorm:"size:36"`
	ContentID     *string                   `gorm:"size:36"`
	Metadata      DBNotificationMetadata    `gorm:"type:json"`
	RetryCount    int                       `gorm:"default:0"`
	CreatedAt     time.Time                 `gorm:"autoCreateTime"`
	UpdatedAt     time.Time                 `gorm:"autoUpdateTime"`
}

type Device struct {
	DeviceToken  string    `gorm:"primaryKey;size:255"`
	UserID       string    `gorm:"column:user_id;type:varchar(36);not null;index" json:"user_id"`
	Platform     string    `gorm:"not null;size:10"`
	RegisteredAt time.Time `gorm:"autoCreateTime"`
	LastActive   time.Time `gorm:"autoCreateTime"`
}

// DBNotificationMetadata implements database serialization for metadata
type DBNotificationMetadata map[string]interface{}

func (nm *DBNotificationMetadata) Scan(value interface{}) error {
	if value == nil {
		*nm = make(DBNotificationMetadata)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into DBNotificationMetadata", value)
	}

	return json.Unmarshal(bytes, nm)
}

func (nm DBNotificationMetadata) Value() (driver.Value, error) {
	return json.Marshal(nm)
}

// Convert to common type
func (nm DBNotificationMetadata) ToCommon() common.NotificationMetadata {
	result := make(common.NotificationMetadata)
	for k, v := range nm {
		result[k] = v
	}
	return result
}

// Convert from common type
func NewDBNotificationMetadata(metadata common.NotificationMetadata) DBNotificationMetadata {
	result := make(DBNotificationMetadata)
	for k, v := range metadata {
		result[k] = v
	}
	return result
}
