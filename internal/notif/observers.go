package notif

import (
	"context"
	"fmt"
	"log"
	"time"

	"gosocial/internal/common"
	"gosocial/internal/dbmysql"

	"firebase.google.com/go/v4/messaging"
)

type DatabaseNotificationObserver struct {
	repo common.NotificationRepository
}

func NewDatabaseNotificationObserver(repo common.NotificationRepository) *DatabaseNotificationObserver {
	return &DatabaseNotificationObserver{
		repo: repo,
	}
}

func (d *DatabaseNotificationObserver) Name() string {
	return "database_observer"
}

func (d *DatabaseNotificationObserver) Update(event common.NotificationEvent) error {
	notification := &dbmysql.Notification{
		ID:            generateID(),
		UserID:        event.UserID,
		Type:          event.Type,
		Header:        event.Header,
		Content:       event.Content,
		ImageURL:      event.ImageURL,
		ScheduledAt:   event.ScheduledAt,
		Priority:      event.Priority,
		Status:        common.StatusPending,
		Metadata:      event.Metadata,
		TriggerUserID: event.TriggerUserID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := d.repo.Create(context.Background(), notification); err != nil {
		return fmt.Errorf("failed to store notification: %w", err)
	}

	return nil
}

type FCMNotificationObserver struct {
	fcmClient  *messaging.Client
	deviceRepo common.DeviceRepository
	notifRepo  common.NotificationRepository
}

func NewFCMNotificationObserver(
	fcmClient *messaging.Client,
	deviceRepo common.DeviceRepository,
	notifRepo common.NotificationRepository,
) *FCMNotificationObserver {
	return &FCMNotificationObserver{
		fcmClient:  fcmClient,
		deviceRepo: deviceRepo,
		notifRepo:  notifRepo,
	}
}

func (f *FCMNotificationObserver) Name() string {
	return "fcm_observer"
}

func (f *FCMNotificationObserver) Update(event common.NotificationEvent) error {
	// Skip if scheduled for future
	if event.ScheduledAt != nil && event.ScheduledAt.After(time.Now()) {
		log.Printf("Notification scheduled for future, skipping FCM: %v", event.ScheduledAt)
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

	tokens := make([]string, len(devicesInterface))
	devices := make([]*dbmysql.Device, len(devicesInterface))
	for i, deviceInterface := range devicesInterface {
		device := deviceInterface.(*dbmysql.Device)
		tokens[i] = device.DeviceToken
		devices[i] = device
	}

	fcmMessage := &messaging.MulticastMessage{ // Create FCM message
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

func (f *FCMNotificationObserver) handleFailedTokens(
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
				}

				log.Printf("Marked invalid token as inactive: %s", device.DeviceToken)
			}
		}
	}
}

type EmailNotificationObserver struct {
	emailService common.EmailService
}

func NewEmailNotificationObserver(emailService common.EmailService) *EmailNotificationObserver {
	return &EmailNotificationObserver{
		emailService: emailService,
	}
}

func (e *EmailNotificationObserver) Name() string {
	return "email_observer"
}

func (e *EmailNotificationObserver) Update(event common.NotificationEvent) error {
	// Only send emails for high priority notifications
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

func generateID() string {
	return fmt.Sprintf("notif_%d", time.Now().UnixNano()) // generateID generates a unique ID for notifications
}
