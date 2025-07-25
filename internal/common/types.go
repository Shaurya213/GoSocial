package common

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type NotificationType string

const (
	FriendRequestType NotificationType = "friend_request"
	PostReactionType  NotificationType = "post_reaction"
	MessageType       NotificationType = "message"
	StoryReactionType NotificationType = "story_reaction"
	SystemType        NotificationType = "system"
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

type NotificationEvent struct {
	Type          NotificationType
	UserID        string
	TriggerUserID *string
	Header        string
	Content       string
	ImageURL      *string
	ScheduledAt   *time.Time
	Priority      int
	Metadata      NotificationMetadata
}

type NotificationResponse struct {
	ID        string               `json:"id"`
	Type      string               `json:"type"`
	Header    string               `json:"header"`
	Content   string               `json:"content"`
	ImageURL  *string              `json:"image_url,omitempty"`
	Status    string               `json:"status"`
	Priority  int                  `json:"priority"`
	Metadata  NotificationMetadata `json:"metadata"`
	CreatedAt time.Time            `json:"created_at"`
	ReadAt    *time.Time           `json:"read_at,omitempty"`
}
