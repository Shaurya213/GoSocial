package dbmysql

import (
	"context"
	"fmt"
	"gosocial/internal/common"
	"time"

	"gorm.io/gorm"
)

type notificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) common.NotificationRepository {
	return &notificationRepository{
		db: db,
	}
}

func (r *notificationRepository) Create(ctx context.Context, notification interface{}) error {
	notif, ok := notification.(*Notification)
	if !ok {
		return fmt.Errorf("invalid notification type")
	}

	if err := r.db.WithContext(ctx).Create(notif).Error; err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	return nil
}

func (r *notificationRepository) ByID(ctx context.Context, id string) (interface{}, error) {
	var notification Notification

	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&notification).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("notification not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	return &notification, nil
}

func (r *notificationRepository) ByUserID(
	ctx context.Context,
	userID string,
	limit, offset int,
) ([]interface{}, error) {
	var notifications []*Notification

	query := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&notifications).Error; err != nil {
		return nil, fmt.Errorf("failed to get user notifications: %w", err)
	}

	result := make([]interface{}, len(notifications)) // Convert to []interface{}
	for i, notif := range notifications {
		result[i] = notif
	}

	return result, nil
}

func (r *notificationRepository) ScheduledNotifications(
	ctx context.Context,
	beforeTime time.Time,
) ([]interface{}, error) {
	var notifications []*Notification

	err := r.db.WithContext(ctx).
		Where("status = ? AND scheduled_at IS NOT NULL AND scheduled_at <= ?",
			string(common.StatusScheduled), beforeTime).
		Order("scheduled_at ASC").
		Find(&notifications).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get scheduled notifications: %w", err)
	}

	result := make([]interface{}, len(notifications))
	for i, notif := range notifications {
		result[i] = notif
	}

	return result, nil
}

func (r *notificationRepository) UpdateStatus(ctx context.Context, id, status string) error {
	result := r.db.WithContext(ctx).
		Model(&Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update notification status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("notification not found: %s", id)
	}

	return nil
}

func (r *notificationRepository) MarkAsRead(ctx context.Context, id, userID string) error {
	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"status":     string(common.StatusRead),
			"read_at":    &now,
			"updated_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to mark notification as read: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("notification not found or access denied: %s", id)
	}

	return nil
}

func (r *notificationRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&Notification{}, "id = ?", id)

	if result.Error != nil {
		return fmt.Errorf("failed to delete notification: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("notification not found: %s", id)
	}

	return nil
}

func (r *notificationRepository) UnreadCount(ctx context.Context, userID string) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&Notification{}).
		Where("user_id = ? AND status != ?", userID, string(common.StatusRead)).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}

