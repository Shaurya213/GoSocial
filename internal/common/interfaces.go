package common

import (
	"context" // provides context for cancellation, deletion, update anything
	"time"
)

type Observer interface {
	Update(event NotificationEvent) error
	Name() string
}

type Subject interface {
	Subscribe(observer Observer)
	Unsubscribe(observer Observer)
	Notify(event NotificationEvent)
	NotifyAsync(event NotificationEvent)
}

type NotificationRepository interface {
	Create(ctx context.Context, notification interface{}) error
	ByID(ctx context.Context, id string) (interface{}, error)
	ByUserID(ctx context.Context, userID string, limit, offset int) ([]interface{}, error)
	ScheduledNotifications(ctx context.Context, beforeTime time.Time) ([]interface{}, error)
	UpdateStatus(ctx context.Context, id, status string) error
	MarkAsRead(ctx context.Context, id, userID string) error
	Delete(ctx context.Context, id string) error
	UnreadCount(ctx context.Context, userID string) (int64, error)
}

type DeviceRepository interface {
	CreateOrUpdate(ctx context.Context, userID, deviceToken, platform string) error
	ActiveByUserID(ctx context.Context, userID string) ([]interface{}, error)
	UpdateTokenStatus(ctx context.Context, token string, isActive bool) error
	DeleteToken(ctx context.Context, token string) error
}

type EmailService interface {
	SendEmail(to, subject, body string) error
}

type EmailData struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	IsHTML  bool     `json:"is_html"`
}
