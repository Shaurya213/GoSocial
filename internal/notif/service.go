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
	"gosocial/internal/user"

	"firebase.google.com/go/v4/messaging"
)

type NotificationSubject struct {
	observers    map[string]common.Observer
	eventChannel chan common.NotificationEvent
	workerPool   int
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	wg           sync.WaitGroup
}

func NewNotificationSubject() *NotificationSubject {
	ctx, cancel := context.WithCancel(context.Background())
	ns := &NotificationSubject{
		observers:    make(map[string]common.Observer),
		eventChannel: make(chan common.NotificationEvent, 1000),
		workerPool:   5, // Default worker pool size
		ctx:          ctx,
		cancel:       cancel,
	}

	for i := 0; i < ns.workerPool; i++ {
		ns.wg.Add(1)
		go ns.processEvents()
	}

	return ns
}

func (ns *NotificationSubject) Subscribe(observer common.Observer) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.observers[observer.Name()] = observer
	log.Printf("Observer %s subscribed", observer.Name())
}

func (ns *NotificationSubject) Unsubscribe(observer common.Observer) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	delete(ns.observers, observer.Name())
	log.Printf("Observer %s unsubscribed", observer.Name())
}

func (ns *NotificationSubject) Notify(event common.NotificationEvent) {
	ns.mu.RLock()
	observers := make([]common.Observer, 0, len(ns.observers))
	for _, obs := range ns.observers {
		observers = append(observers, obs)
	}
	ns.mu.RUnlock()

	for _, observer := range observers {
		if err := observer.Update(event); err != nil {
			log.Printf("Observer %s update failed: %v", observer.Name(), err)
		}
	}
}

func (ns *NotificationSubject) NotifyAsync(event common.NotificationEvent) {
	select {
	case ns.eventChannel <- event:
	case <-ns.ctx.Done():
		return
	default:
		log.Printf("Notification channel full, dropping event: %s", event.Type)
	}
}

func (ns *NotificationSubject) processEvents() {
	defer ns.wg.Done()
	for {
		select {
		case event := <-ns.eventChannel:
			ns.Notify(event)
		case <-ns.ctx.Done():
			return
		}
	}
}

func (ns *NotificationSubject) Shutdown() {
	ns.cancel()
	close(ns.eventChannel)
	ns.wg.Wait()
	log.Println("NotificationSubject shutdown complete")
}

type NotificationService struct {
	config       *config.Config
	repo         common.NotificationRepository
	deviceRepo   user.DeviceRepository
	fcmClient    *messaging.Client
	emailService common.EmailService
	subject      *NotificationSubject
}

func NewNotificationService(config *config.Config, repo common.NotificationRepository, deviceRepo user.DeviceRepository, fcmClient *messaging.Client, emailService common.EmailService) *NotificationService {
	service := &NotificationService{
		config:       config,
		repo:         repo,
		deviceRepo:   deviceRepo,
		fcmClient:    fcmClient,
		emailService: emailService,
		subject:      NewNotificationSubject(),
	}

	if fcmClient != nil {
		service.subject.Subscribe(NewFCMObserver(fcmClient, deviceRepo))
	}
	if emailService != nil {
		service.subject.Subscribe(NewEmailObserver(emailService))
	}

	go service.processScheduledNotifications()

	return service
}

func (s *NotificationService) SendNotification(ctx context.Context, event common.NotificationEvent) error {

	if err := s.validateEvent(event); err != nil {
		return fmt.Errorf("invalid notification event: %w", err)
	}

	s.subject.NotifyAsync(event)

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

	notification := &dbmysql.Notification{
		// ID is omitted so the database can auto-increment it
		UserID:        uint64(event.UserID),
		Type:          event.Type,
		Header:        event.Header,
		Content:       event.Content,
		ImageURL:      event.ImageURL,
		ScheduledAt:   event.ScheduledAt,
		Priority:      event.Priority,
		Status:        common.StatusScheduled,
		TriggerUserID: event.TriggerUserID,
		Metadata:      dbmysql.NewDBNotificationMetadata(event.Metadata),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return fmt.Errorf("failed to create scheduled notification: %w", err)
	}

	log.Printf("Notification scheduled: type=%s, user=%s, scheduled_at=%v",
		event.Type, event.UserID, event.ScheduledAt)
	return nil
}

func (s *NotificationService) GetUserNotifications(ctx context.Context, userID uint64, limit, offset int) ([]*dbmysql.Notification, error) {
	results, err := s.repo.ByUserID(ctx, uint(userID), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}

	notifications := make([]*dbmysql.Notification, len(results))
	for i, result := range results {
		if notif, ok := result.(*dbmysql.Notification); ok {
			notifications[i] = notif
		} else {
			return nil, fmt.Errorf("invalid notification type in result")
		}
	}

	return notifications, nil
}

func (s *NotificationService) MarkAsRead(ctx context.Context, notificationID uint, userID uint) error {
	return s.repo.MarkAsRead(ctx, notificationID, userID)
}

func (s *NotificationService) RegisterDeviceToken(ctx context.Context, userID uint, deviceToken, platform string) error {
	return s.deviceRepo.CreateOrUpdate(ctx, uint64(userID), deviceToken, platform)
}

func (s *NotificationService) SendFriendRequestNotification(ctx context.Context, fromUserID uint, toUserID uint, fromUsername string) error {
	fromUserIDStr := fmt.Sprintf("%d", fromUserID)
	event := common.NotificationEvent{
		Type:          common.FriendRequestType,
		UserID:        toUserID,
		TriggerUserID: &fromUserIDStr,
		Header:        "Friend Request",
		Content:       fmt.Sprintf("%s sent you a friend request", fromUsername),
		Priority:      3,
		Metadata: common.NotificationMetadata{
			"from_user_id":  fromUserID,
			"from_username": fromUsername,
			"action_type":   "friend_request",
		},
	}

	return s.SendNotification(ctx, event)
}

func (s *NotificationService) processScheduledNotifications() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		notificationsInterface, err := s.repo.ScheduledNotifications(ctx, time.Now())
		if err != nil {
			log.Printf("Failed to get scheduled notifications: %v", err)
			continue
		}

		for _, notifInterface := range notificationsInterface {
			if notif, ok := notifInterface.(*dbmysql.Notification); ok {
				event := common.NotificationEvent{
					Type:          notif.Type,
					UserID:        uint(notif.UserID),
					TriggerUserID: notif.TriggerUserID,
					Header:        notif.Header,
					Content:       notif.Content,
					ImageURL:      notif.ImageURL,
					Priority:      notif.Priority,
					Metadata:      notif.Metadata.ToCommon(),
				}

				s.subject.NotifyAsync(event)

				// Update status to sent
				if err := s.repo.UpdateStatus(ctx, notif.ID, string(common.StatusSent)); err != nil {
					log.Printf("Failed to update notification status: %v", err)
				}
			}
		}

		if len(notificationsInterface) > 0 {
			log.Printf("Processed %d scheduled notifications", len(notificationsInterface))
		}
	}
}

func (s *NotificationService) validateEvent(event common.NotificationEvent) error {
	if event.UserID == 0 {
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
	s.subject.Shutdown()
	log.Println("NotificationService shutdown complete")
}

var _ NotificationServiceInterface = (*NotificationService)(nil)
