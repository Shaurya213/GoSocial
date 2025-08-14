package common

import (
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

type NotificationEvent struct {
	Type          NotificationType
	UserID        uint
	TriggerUserID *string
	Header        string
	Content       string
	ImageURL      *string
	ScheduledAt   *time.Time
	Priority      int
	Metadata      NotificationMetadata
}

type NotificationResponse struct {
	ID        uint                 `json:"id"`
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
