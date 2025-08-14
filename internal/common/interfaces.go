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
	ByID(ctx context.Context, id uint) (interface{}, error)
	ByUserID(ctx context.Context, userID uint, limit, offset int) ([]interface{}, error)
	ScheduledNotifications(ctx context.Context, beforeTime time.Time) ([]interface{}, error)
	UpdateStatus(ctx context.Context, id uint, status string) error
	MarkAsRead(ctx context.Context, id uint, userID uint) error
	Delete(ctx context.Context, id uint) error
	UnreadCount(ctx context.Context, userID uint) (int64, error)
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
