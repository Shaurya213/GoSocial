package notif

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"gosocial/internal/common"
	"gosocial/internal/config"
	"gosocial/internal/dbmysql"

	"firebase.google.com/go/v4/messaging"
)

type NotificationManager struct {
	observers    map[string]common.Observer
	eventChannel chan common.NotificationEvent
	workerPool   int
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	wg           sync.WaitGroup
}

func NewNotificationManager(workerPoolSize int) *NotificationManager {
	ctx, cancel := context.WithCancel(context.Background())

	nm := &NotificationManager{
		observers:    make(map[string]common.Observer),
		eventChannel: make(chan common.NotificationEvent, 1000),
		workerPool:   workerPoolSize,
		ctx:          ctx,
		cancel:       cancel,
	}

	for i := 0; i < workerPoolSize; i++ {
		nm.wg.Add(1)
		go nm.processEvents()
	}

	return nm
}

func (nm *NotificationManager) Subscribe(observer common.Observer) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.observers[observer.Name()] = observer
	log.Printf("Observer %s subscribed", observer.Name())
}

func (nm *NotificationManager) Unsubscribe(observer common.Observer) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	delete(nm.observers, observer.Name())
	log.Printf("Observer %s unsubscribed", observer.Name())
}

func (nm *NotificationManager) Notify(event common.NotificationEvent) {
	nm.mu.RLock()
	observers := make([]common.Observer, 0, len(nm.observers))
	for _, obs := range nm.observers {
		observers = append(observers, obs)
	}
	nm.mu.RUnlock()

	for _, observer := range observers {
		if err := observer.Update(event); err != nil {
			log.Printf("Observer %s update failed: %v", observer.Name(), err)
		}
	}
}

func (nm *NotificationManager) NotifyAsync(event common.NotificationEvent) {
	select {
	case nm.eventChannel <- event:

	case <-nm.ctx.Done():
		return
	default:
		log.Printf("Notification channel full, dropping event: %s", event.Type)
	}
}

func (nm *NotificationManager) processEvents() {
	defer nm.wg.Done()

	for {
		select {
		case event := <-nm.eventChannel:
			nm.Notify(event)
		case <-nm.ctx.Done():
			return
		}
	}
}

func (nm *NotificationManager) Shutdown() {
	nm.cancel()
	close(nm.eventChannel)
	nm.wg.Wait()
	log.Println("NotificationManager shutdown complete")
}

type NotificationService struct {
	manager      *NotificationManager
	repo         common.NotificationRepository
	deviceRepo   common.DeviceRepository
	emailService common.EmailService
}

func NewNotificationService(
	cfg *config.Config,
	repo common.NotificationRepository,
	deviceRepo common.DeviceRepository,
	fcmClient *messaging.Client,
	emailService common.EmailService,
) *NotificationService {

	manager := NewNotificationManager(cfg.Notification.Workers)

	dbObserver := NewDatabaseNotificationObserver(repo)
	manager.Subscribe(dbObserver)

	if fcmClient != nil {
		fcmObserver := NewFCMNotificationObserver(fcmClient, deviceRepo, repo)
		manager.Subscribe(fcmObserver)
	}

	if emailService != nil {
		emailObserver := NewEmailNotificationObserver(emailService)
		manager.Subscribe(emailObserver)
	}

	service := &NotificationService{
		manager:      manager,
		repo:         repo,
		deviceRepo:   deviceRepo,
		emailService: emailService,
	}

	go service.processScheduledNotifications()

	return service
}

func (s *NotificationService) SendNotification(ctx context.Context, event common.NotificationEvent) error {
	// Validate event
	if err := s.validateEvent(event); err != nil {
		return fmt.Errorf("invalid notification event: %w", err)
	}

	s.manager.Notify(event)

	log.Printf("Notification sent: type=%s, user=%s", event.Type, event.UserID)
	return nil
}

func (s *NotificationService) ScheduleNotification(ctx context.Context, event common.NotificationEvent) error {
	if event.ScheduledAt == nil {
		return fmt.Errorf("scheduled_at is required for scheduled notifications")
	}

	if event.ScheduledAt.Before(time.Now()) {
		return fmt.Errorf("scheduled_at must be in the future")
	}

	if err := s.validateEvent(event); err != nil {
		return fmt.Errorf("invalid notification event: %w", err)
	}

	event.Priority = 1 // Lower priority for scheduled notifications
	s.manager.NotifyAsync(event)

	log.Printf("Notification scheduled: type=%s, user=%s, scheduled_at=%v",
		event.Type, event.UserID, event.ScheduledAt)
	return nil
}

func (s *NotificationService) SendFriendRequestNotification(
	ctx context.Context,
	fromUserID, toUserID, fromUserHandle string,
) error {
	event := common.NotificationEvent{
		Type:          common.FriendRequestType,
		UserID:        toUserID,
		TriggerUserID: &fromUserID,
		Header:        "New Friend Request",
		Content:       fmt.Sprintf("%s sent you a friend request", fromUserHandle),
		Priority:      3,
		Metadata: common.NotificationMetadata{
			"from_user_id": fromUserID,
			"action":       "friend_request",
		},
	}

	return s.SendNotification(ctx, event)
}

func (s *NotificationService) SendReactionNotification(
	ctx context.Context,
	contentID, contentAuthorID, reactorUserID, reactorHandle, reactionType string,
) error {

	if contentAuthorID == reactorUserID {
		return nil
	}

	event := common.NotificationEvent{
		Type:          common.PostReactionType,
		UserID:        contentAuthorID,
		TriggerUserID: &reactorUserID,
		Header:        "New Reaction",
		Content:       fmt.Sprintf("%s reacted to your post", reactorHandle),
		Priority:      2,
		Metadata: common.NotificationMetadata{
			"content_id":     contentID,
			"reaction_type":  reactionType,
			"reactor_handle": reactorHandle,
		},
	}

	return s.SendNotification(ctx, event)
}

func (s *NotificationService) SendMessageNotification(
	ctx context.Context,
	conversationID, recipientUserID, senderUserID, senderHandle, messagePreview string,
) error {
	event := common.NotificationEvent{
		Type:          common.MessageType,
		UserID:        recipientUserID,
		TriggerUserID: &senderUserID,
		Header:        fmt.Sprintf("Message from %s", senderHandle),
		Content:       messagePreview,
		Priority:      4, // High priority for messages
		Metadata: common.NotificationMetadata{
			"conversation_id": conversationID,
			"sender_handle":   senderHandle,
		},
	}

	return s.SendNotification(ctx, event)
}

func (s *NotificationService) GetUserNotifications(
	ctx context.Context,
	userID string,
	limit, offset int,
) ([]*common.NotificationResponse, error) {
	notificationsInterface, err := s.repo.ByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}

	responses := make([]*common.NotificationResponse, len(notificationsInterface))
	for i, notifInterface := range notificationsInterface {
		notif := notifInterface.(*dbmysql.Notification)
		responses[i] = &common.NotificationResponse{
			ID:        notif.ID,
			Type:      string(notif.Type),
			Header:    notif.Header,
			Content:   notif.Content,
			ImageURL:  notif.ImageURL,
			Status:    string(notif.Status),
			Priority:  notif.Priority,
			Metadata:  notif.Metadata,
			CreatedAt: notif.CreatedAt,
			ReadAt:    notif.ReadAt,
		}
	}

	return responses, nil
}

func (s *NotificationService) MarkAsRead(ctx context.Context, notificationID, userID string) error {
	return s.repo.MarkAsRead(ctx, notificationID, userID)
}

func (s *NotificationService) RegisterDeviceToken(
	ctx context.Context,
	userID, deviceToken, platform string,
) error {
	return s.deviceRepo.CreateOrUpdate(ctx, userID, deviceToken, platform)
}

func (s *NotificationService) processScheduledNotifications() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()

		notificationsInterface, err := s.repo.
			ScheduledNotifications(ctx, time.Now())
		if err != nil {
			log.Printf("Failed to get scheduled notifications: %v", err)
			continue
		}

		for _, notifInterface := range notificationsInterface {
			notif := notifInterface.(*dbmysql.Notification)

			event := common.NotificationEvent{
				Type:          common.NotificationType(notif.Type),
				UserID:        notif.UserID,
				TriggerUserID: notif.TriggerUserID,
				Header:        notif.Header,
				Content:       notif.Content,
				ImageURL:      notif.ImageURL,
				Priority:      notif.Priority,
				Metadata:      notif.Metadata,
			}

			s.manager.NotifyAsync(event)

			if err := s.repo.UpdateStatus(ctx, notif.ID, string(common.StatusSent)); err != nil {
				log.Printf("Failed to update notification status: %v", err)
			}
		}

		if len(notificationsInterface) > 0 {
			log.Printf("Processed %d scheduled notifications", len(notificationsInterface))
		}
	}
}

func (s *NotificationService) validateEvent(event common.NotificationEvent) error {
	if event.UserID == "" {
		return fmt.Errorf("user_id is required")
	}

	if event.Header == "" {
		return fmt.Errorf("header is required")
	}

	if event.Content == "" {
		return fmt.Errorf("content is required")
	}

	if event.Priority < 1 || event.Priority > 5 {
		return fmt.Errorf("priority must be between 1 and 5")
	}

	return nil
}

func (s *NotificationService) Shutdown() {
	s.manager.Shutdown()
	log.Println("NotificationService shutdown complete")
}
