package notif

import (
	"context"
	"errors"
	"testing"
	"time"

	"gosocial/internal/common"

	"firebase.google.com/go/v4/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock Firebase Client for FCM testing
type MockFCMClient struct {
	mock.Mock
}

func (m *MockFCMClient) SendMulticast(ctx context.Context, message *messaging.MulticastMessage) (*messaging.BatchResponse, error) {
	args := m.Called(ctx, message)
	return args.Get(0).(*messaging.BatchResponse), args.Error(1)
}

// Mock Device Repository for observer testing
type MockDeviceRepositoryForObserver struct {
	mock.Mock
}

func (m *MockDeviceRepositoryForObserver) CreateOrUpdate(ctx context.Context, userID, deviceToken, platform string) error {
	args := m.Called(ctx, userID, deviceToken, platform)
	return args.Error(0)
}

func (m *MockDeviceRepositoryForObserver) ActiveByUserID(ctx context.Context, userID string) ([]interface{}, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *MockDeviceRepositoryForObserver) UpdateTokenStatus(ctx context.Context, token string, isActive bool) error {
	args := m.Called(ctx, token, isActive)
	return args.Error(0)
}

func (m *MockDeviceRepositoryForObserver) DeleteToken(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

// Mock Email Service for observer testing
type MockEmailServiceForObserver struct {
	mock.Mock
}

func (m *MockEmailServiceForObserver) SendEmail(to, subject, body string) error {
	args := m.Called(to, subject, body)
	return args.Error(0)
}

// Mock Notification Repository for observer testing
type MockNotificationRepositoryForObserver struct {
	mock.Mock
}

func (m *MockNotificationRepositoryForObserver) Create(ctx context.Context, notification interface{}) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *MockNotificationRepositoryForObserver) ByID(ctx context.Context, id string) (interface{}, error) {
	args := m.Called(ctx, id)
	return args.Get(0), args.Error(1)
}

func (m *MockNotificationRepositoryForObserver) ByUserID(ctx context.Context, userID string, limit, offset int) ([]interface{}, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *MockNotificationRepositoryForObserver) ScheduledNotifications(ctx context.Context, beforeTime time.Time) ([]interface{}, error) {
	args := m.Called(ctx, beforeTime)
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *MockNotificationRepositoryForObserver) UpdateStatus(ctx context.Context, id, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockNotificationRepositoryForObserver) MarkAsRead(ctx context.Context, id, userID string) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockNotificationRepositoryForObserver) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockNotificationRepositoryForObserver) UnreadCount(ctx context.Context, userID string) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

// FCMObserver Tests
func TestNewFCMObserver(t *testing.T) {
	mockDeviceRepo := &MockDeviceRepositoryForObserver{}
	var fcmClient *messaging.Client = nil

	// FIXED: Use the actual NewFCMObserver from observers.go
	observer := NewFCMObserver(fcmClient, mockDeviceRepo)

	assert.NotNil(t, observer)
	// FIXED: Access fields through the actual FCMObserver struct
	// Note: These fields are private, so we test behavior instead
	assert.Equal(t, "fcm_observer", observer.Name())
}

func TestFCMObserver_Name(t *testing.T) {
	mockDeviceRepo := &MockDeviceRepositoryForObserver{}
	observer := NewFCMObserver(nil, mockDeviceRepo)

	name := observer.Name()
	assert.Equal(t, "fcm_observer", name)
}

func TestFCMObserver_Update_ScheduledNotification(t *testing.T) {
	mockDeviceRepo := &MockDeviceRepositoryForObserver{}
	observer := NewFCMObserver(nil, mockDeviceRepo)

	futureTime := time.Now().Add(1 * time.Hour)
	event := common.NotificationEvent{
		Type:        "test",
		UserID:      "user1",
		Header:      "Test",
		Content:     "Test content",
		Priority:    3,
		ScheduledAt: &futureTime,
		Metadata:    common.NotificationMetadata{},
	}

	err := observer.Update(event)

	// Should skip FCM for scheduled notifications in the future
	assert.NoError(t, err)
}

func TestFCMObserver_Update_NilFCMClient(t *testing.T) {
	mockDeviceRepo := &MockDeviceRepositoryForObserver{}
	observer := NewFCMObserver(nil, mockDeviceRepo) // nil FCM client

	event := common.NotificationEvent{
		Type:     "test",
		UserID:   "user1",
		Header:   "Test",
		Content:  "Test content",
		Priority: 3,
		Metadata: common.NotificationMetadata{},
	}

	err := observer.Update(event)

	// Should skip FCM when client is nil
	assert.NoError(t, err)
}

// FIXED: This test was failing because FCM client is nil, so it never calls the repository
func TestFCMObserver_Update_NoActiveDevices(t *testing.T) {
	mockDeviceRepo := &MockDeviceRepositoryForObserver{}
	// Create a non-nil FCM client for this test to actually reach the repository call
	var fcmClient *messaging.Client = &messaging.Client{} // Non-nil but empty client
	observer := NewFCMObserver(fcmClient, mockDeviceRepo)

	event := common.NotificationEvent{
		Type:     "test",
		UserID:   "user1",
		Header:   "Test",
		Content:  "Test content",
		Priority: 3,
		Metadata: common.NotificationMetadata{},
	}

	// Mock no active devices
	mockDeviceRepo.On("ActiveByUserID", mock.Anything, "user1").Return([]interface{}{}, nil)

	err := observer.Update(event)

	assert.NoError(t, err)
	mockDeviceRepo.AssertExpectations(t)
}

// FIXED: Same issue - need non-nil FCM client to reach repository error
func TestFCMObserver_Update_DeviceRepoError(t *testing.T) {
	mockDeviceRepo := &MockDeviceRepositoryForObserver{}
	// Create a non-nil FCM client
	var fcmClient *messaging.Client = &messaging.Client{}
	observer := NewFCMObserver(fcmClient, mockDeviceRepo)

	event := common.NotificationEvent{
		Type:     "test",
		UserID:   "user1",
		Header:   "Test",
		Content:  "Test content",
		Priority: 3,
		Metadata: common.NotificationMetadata{},
	}

	// Mock repository error
	mockDeviceRepo.On("ActiveByUserID", mock.Anything, "user1").Return([]interface{}{}, errors.New("db error"))

	err := observer.Update(event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get devices")
	mockDeviceRepo.AssertExpectations(t)
}

func TestFCMObserver_Update_InvalidDeviceType(t *testing.T) {
	mockDeviceRepo := &MockDeviceRepositoryForObserver{}
	var fcmClient *messaging.Client = &messaging.Client{}
	observer := NewFCMObserver(fcmClient, mockDeviceRepo)

	event := common.NotificationEvent{
		Type:     "test",
		UserID:   "user1",
		Header:   "Test",
		Content:  "Test content",
		Priority: 3,
		Metadata: common.NotificationMetadata{},
	}

	// Mock devices with invalid type
	devices := []interface{}{
		"invalid_device_type",
		123,
	}
	mockDeviceRepo.On("ActiveByUserID", mock.Anything, "user1").Return(devices, nil)

	err := observer.Update(event)

	// Should handle invalid device types gracefully
	assert.NoError(t, err)
	mockDeviceRepo.AssertExpectations(t)
}

// EmailObserver Tests
func TestNewEmailObserver(t *testing.T) {
	mockEmailService := &MockEmailServiceForObserver{}

	observer := NewEmailObserver(mockEmailService)

	assert.NotNil(t, observer)
	assert.Equal(t, "email_observer", observer.Name())
}

func TestEmailObserver_Name(t *testing.T) {
	mockEmailService := &MockEmailServiceForObserver{}
	observer := NewEmailObserver(mockEmailService)

	name := observer.Name()
	assert.Equal(t, "email_observer", name)
}

func TestEmailObserver_Update_LowPriority(t *testing.T) {
	mockEmailService := &MockEmailServiceForObserver{}
	observer := NewEmailObserver(mockEmailService)

	event := common.NotificationEvent{
		Type:     "test",
		UserID:   "user1",
		Header:   "Test",
		Content:  "Test content",
		Priority: 3, // Below threshold of 4
		Metadata: common.NotificationMetadata{
			"email": "user@example.com",
		},
	}

	err := observer.Update(event)

	// Should skip email for low priority notifications
	assert.NoError(t, err)
	mockEmailService.AssertNotCalled(t, "SendEmail")
}

func TestEmailObserver_Update_SuccessfulEmail(t *testing.T) {
	mockEmailService := &MockEmailServiceForObserver{}
	observer := NewEmailObserver(mockEmailService)

	event := common.NotificationEvent{
		Type:     "urgent",
		UserID:   "user1",
		Header:   "Urgent Notification",
		Content:  "This is urgent content",
		Priority: 5,
		Metadata: common.NotificationMetadata{
			"email": "user@example.com",
		},
	}

	expectedSubject := "GoSocial Notification: Urgent Notification"
	expectedBody := "This is urgent content"
	expectedTo := "user@example.com"

	mockEmailService.On("SendEmail", expectedTo, expectedSubject, expectedBody).Return(nil)

	err := observer.Update(event)

	assert.NoError(t, err)
	mockEmailService.AssertExpectations(t)
}

// DatabaseObserver Tests
func TestNewDatabaseObserver(t *testing.T) {
	mockRepo := &MockNotificationRepositoryForObserver{}

	observer := NewDatabaseObserver(mockRepo)

	assert.NotNil(t, observer)
	assert.Equal(t, "database_observer", observer.Name())
}

func TestDatabaseObserver_Update_Successful(t *testing.T) {
	mockRepo := &MockNotificationRepositoryForObserver{}
	observer := NewDatabaseObserver(mockRepo)

	event := common.NotificationEvent{
		Type:          "test",
		UserID:        "user1",
		TriggerUserID: func() *string { s := "trigger1"; return &s }(),
		Header:        "Test Notification",
		Content:       "Test content",
		ImageURL:      func() *string { s := "http://example.com/image.jpg"; return &s }(),
		Priority:      3,
		Metadata: common.NotificationMetadata{
			"key1": "value1",
		},
	}

	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*dbmysql.Notification")).Return(nil)

	err := observer.Update(event)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestDatabaseObserver_Update_RepositoryError(t *testing.T) {
	mockRepo := &MockNotificationRepositoryForObserver{}
	observer := NewDatabaseObserver(mockRepo)

	event := common.NotificationEvent{
		Type:     "test",
		UserID:   "user1",
		Header:   "Test Notification",
		Content:  "Test content",
		Priority: 3,
		Metadata: common.NotificationMetadata{},
	}

	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*dbmysql.Notification")).Return(errors.New("database error"))

	err := observer.Update(event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store notification")
	mockRepo.AssertExpectations(t)
}

// Integration test
func TestObservers_Integration(t *testing.T) {
	t.Run("all observers process same event", func(t *testing.T) {
		// Setup all observers
		mockDeviceRepo := &MockDeviceRepositoryForObserver{}
		mockEmailService := &MockEmailServiceForObserver{}
		mockNotifRepo := &MockNotificationRepositoryForObserver{}

		// Use real constructors
		fcmObserver := NewFCMObserver(nil, mockDeviceRepo)
		emailObserver := NewEmailObserver(mockEmailService)
		dbObserver := NewDatabaseObserver(mockNotifRepo)

		event := common.NotificationEvent{
			Type:     "urgent",
			UserID:   "user1",
			Header:   "Integration Test",
			Content:  "Integration test content",
			Priority: 5,
			Metadata: common.NotificationMetadata{
				"email": "user@example.com",
			},
		}

		// Setup expectations - FCM will skip due to nil client
		mockEmailService.On("SendEmail", "user@example.com", "GoSocial Notification: Integration Test", "Integration test content").Return(nil)
		mockNotifRepo.On("Create", mock.Anything, mock.AnythingOfType("*dbmysql.Notification")).Return(nil)

		// Execute
		err1 := fcmObserver.Update(event)
		err2 := emailObserver.Update(event)
		err3 := dbObserver.Update(event)

		// Assert
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NoError(t, err3)

		mockEmailService.AssertExpectations(t)
		mockNotifRepo.AssertExpectations(t)
	})
}

// Benchmark Tests
func BenchmarkFCMObserver_Update(b *testing.B) {
	mockDeviceRepo := &MockDeviceRepositoryForObserver{}
	observer := NewFCMObserver(nil, mockDeviceRepo)

	event := common.NotificationEvent{
		Type:     "benchmark",
		UserID:   "user1",
		Header:   "Benchmark",
		Content:  "Benchmark content",
		Priority: 3,
		Metadata: common.NotificationMetadata{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		observer.Update(event)
	}
}
