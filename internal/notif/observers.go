package notif

import (
	"context"
	"fmt"
	"log"
	"time"

	"gosocial/internal/common"
	"gosocial/internal/dbmysql"
	"gosocial/internal/user"

	"firebase.google.com/go/v4/messaging"
	"github.com/google/uuid"
)

// FCMObserver handles Firebase Cloud Messaging notifications
type FCMObserver struct {
	fcmClient  *messaging.Client
	deviceRepo user.DeviceRepository
}

func NewFCMObserver(
	fcmClient *messaging.Client,
	deviceRepo user.DeviceRepository,
) *FCMObserver {
	return &FCMObserver{
		fcmClient:  fcmClient,
		deviceRepo: deviceRepo,
	}
}

func (f *FCMObserver) Name() string {
	return "fcm_observer"
}

func (f *FCMObserver) Update(event common.NotificationEvent) error {
	if event.ScheduledAt != nil && event.ScheduledAt.After(time.Now()) {
		log.Printf("Notification scheduled for future, skipping FCM: %v", event.ScheduledAt)
		return nil
	}

	if f.fcmClient == nil {
		log.Printf("FCM client not available, skipping FCM notification")
		return nil
	}

	devicesInterface, err := f.deviceRepo.ActiveByUserID(context.Background(), event.UserID)
	if err != nil {
		return fmt.Errorf("failed to get devices: %w", err)
	}

	if len(devicesInterface) == 0 {
		log.Printf("No active devices found for user: %s", event.UserID)
		return nil
	}

	tokens := make([]string, 0, len(devicesInterface))
	devices := make([]*dbmysql.Device, 0, len(devicesInterface))
	for _, deviceInterface := range devicesInterface {
		if device, ok := deviceInterface.(*dbmysql.Device); ok {
			tokens = append(tokens, device.DeviceToken)
			devices = append(devices, device)
		}
	}

	if len(tokens) == 0 {
		log.Printf("No valid device tokens found for user: %s", event.UserID)
		return nil
	}

	fcmMessage := &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: event.Header,
			Body:  event.Content,
		},
		Data: map[string]string{
			"type":    string(event.Type),
			"user_id": event.UserID,
		},
		Tokens: tokens,
	}

	if event.ImageURL != nil {
		fcmMessage.Notification.ImageURL = *event.ImageURL
	}

	if event.Metadata != nil {
		for key, value := range event.Metadata {
			if strValue, ok := value.(string); ok {
				fcmMessage.Data[key] = strValue
			}
		}
	}

	response, err := f.fcmClient.SendMulticast(context.Background(), fcmMessage)
	if err != nil {
		return fmt.Errorf("failed to send FCM: %w", err)
	}

	f.handleFailedTokens(response, devices)
	log.Printf("FCM notification sent: %d success, %d failure",
		response.SuccessCount, response.FailureCount)
	return nil
}

func (f *FCMObserver) handleFailedTokens(
	response *messaging.BatchResponse,
	devices []*dbmysql.Device,
) {
	for i, result := range response.Responses {
		if !result.Success && i < len(devices) {
			device := devices[i]
			if messaging.IsRegistrationTokenNotRegistered(result.Error) ||
				messaging.IsInvalidArgument(result.Error) {
				// Mark token as inactive
				if err := f.deviceRepo.UpdateTokenStatus(
					context.Background(),
					device.DeviceToken,
					false,
				); err != nil {
					log.Printf("Failed to update token status: %v", err)
				} else {
					log.Printf("Marked invalid token as inactive: %s", device.DeviceToken)
				}
			}
		}
	}
}

type EmailObserver struct {
	emailService common.EmailService
}

func NewEmailObserver(emailService common.EmailService) *EmailObserver {
	return &EmailObserver{
		emailService: emailService,
	}
}

func (e *EmailObserver) Name() string {
	return "email_observer"
}

func (e *EmailObserver) Update(event common.NotificationEvent) error {
	if event.Priority < 4 {
		return nil
	}

	email, ok := event.Metadata["email"].(string)
	if !ok || email == "" {
		return nil // No email provided
	}

	subject := fmt.Sprintf("GoSocial Notification: %s", event.Header)
	body := event.Content
	if err := e.emailService.SendEmail(email, subject, body); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Email notification sent to: %s", email)
	return nil
}

type DatabaseObserver struct {
	repo common.NotificationRepository
}

func NewDatabaseObserver(repo common.NotificationRepository) *DatabaseObserver {
	return &DatabaseObserver{
		repo: repo,
	}
}

func (d *DatabaseObserver) Name() string {
	return "database_observer"
}

func (d *DatabaseObserver) Update(event common.NotificationEvent) error {
	notification := &dbmysql.Notification{
		ID:            uuid.New().String(),
		UserID:        event.UserID,
		Type:          event.Type,
		Header:        event.Header,
		Content:       event.Content,
		ImageURL:      event.ImageURL,
		ScheduledAt:   event.ScheduledAt,
		Priority:      event.Priority,
		Status:        common.StatusPending,
		TriggerUserID: event.TriggerUserID,
		Metadata:      dbmysql.NewDBNotificationMetadata(event.Metadata),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := d.repo.Create(context.Background(), notification); err != nil {
		return fmt.Errorf("failed to store notification: %w", err)
	}

	return nil
}
