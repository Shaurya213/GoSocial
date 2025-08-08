package notif

import (
	"context"
	"errors"
	"testing"
	"time"

	pb "gosocial/api/v1"
	"gosocial/internal/common"
	"gosocial/internal/config"
	"gosocial/internal/dbmysql"

	"firebase.google.com/go/v4/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FIXED: Complete MockNotificationService implementing NotificationServiceInterface
type MockNotificationServiceForHandler struct {
	mock.Mock
}

func (m *MockNotificationServiceForHandler) SendNotification(ctx context.Context, event common.NotificationEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockNotificationServiceForHandler) ScheduleNotification(ctx context.Context, event common.NotificationEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockNotificationServiceForHandler) GetUserNotifications(ctx context.Context, userID string, limit, offset int) ([]*dbmysql.Notification, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]*dbmysql.Notification), args.Error(1)
}

func (m *MockNotificationServiceForHandler) MarkAsRead(ctx context.Context, notificationID, userID string) error {
	args := m.Called(ctx, notificationID, userID)
	return args.Error(0)
}

func (m *MockNotificationServiceForHandler) RegisterDeviceToken(ctx context.Context, userID, deviceToken, platform string) error {
	args := m.Called(ctx, userID, deviceToken, platform)
	return args.Error(0)
}

func (m *MockNotificationServiceForHandler) SendFriendRequestNotification(ctx context.Context, fromUserID, toUserID, fromUsername string) error {
	args := m.Called(ctx, fromUserID, toUserID, fromUsername)
	return args.Error(0)
}

func (m *MockNotificationServiceForHandler) Shutdown() {
	m.Called()
}

// FIXED: Complete MockDeviceRepository
type MockDeviceRepositoryForHandler struct {
	mock.Mock
}

func (m *MockDeviceRepositoryForHandler) CreateOrUpdate(ctx context.Context, userID, deviceToken, platform string) error {
	args := m.Called(ctx, userID, deviceToken, platform)
	return args.Error(0)
}

func (m *MockDeviceRepositoryForHandler) ActiveByUserID(ctx context.Context, userID string) ([]interface{}, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *MockDeviceRepositoryForHandler) UpdateTokenStatus(ctx context.Context, token string, isActive bool) error {
	args := m.Called(ctx, token, isActive)
	return args.Error(0)
}

func (m *MockDeviceRepositoryForHandler) DeleteToken(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

// FIXED: Test helper functions
func createTestGRPCHandler() *GRPCHandler {
	mockService := &MockNotificationServiceForHandler{}
	mockConfig := &config.Config{}
	mockDeviceRepo := &MockDeviceRepositoryForHandler{}

	return NewGRPCHandler(mockService, mockConfig, nil, mockDeviceRepo)
}

func TestNewNotificationHandler(t *testing.T) {
	mockService := &MockNotificationServiceForHandler{}
	mockConfig := &config.Config{}
	mockDeviceRepo := &MockDeviceRepositoryForHandler{}

	// This should now work without type errors
	handler := NewNotificationHandler(mockService, mockConfig, nil, mockDeviceRepo)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.GRPC)
	assert.Equal(t, mockService, handler.GRPC.service)
	assert.Equal(t, mockConfig, handler.GRPC.config)
	assert.Equal(t, mockDeviceRepo, handler.GRPC.deviceRepo)
}

func TestNewGRPCHandler(t *testing.T) {
	mockService := &MockNotificationServiceForHandler{}
	mockConfig := &config.Config{}
	mockDeviceRepo := &MockDeviceRepositoryForHandler{}
	var fcmClient *messaging.Client = nil

	handler := NewGRPCHandler(mockService, mockConfig, fcmClient, mockDeviceRepo)

	assert.NotNil(t, handler)
	assert.Equal(t, mockService, handler.service)
	assert.Equal(t, mockConfig, handler.config)
	assert.Equal(t, fcmClient, handler.fcmClient)
	assert.Equal(t, mockDeviceRepo, handler.deviceRepo)
}

func TestGRPCHandler_SendNotification(t *testing.T) {
	t.Run("successful send notification", func(t *testing.T) {
		mockService := &MockNotificationServiceForHandler{}
		handler := &GRPCHandler{
			service: mockService,
			config:  &config.Config{},
		}

		req := &pb.SendNotificationRequest{
			UserId:  "user123",
			Title:   "Test Notification",
			Message: "This is a test message",
			Type:    "test",
			Data: map[string]string{
				"key1": "value1",
			},
		}

		expectedEvent := common.NotificationEvent{
			Type:     common.NotificationType("test"),
			UserID:   "user123",
			Header:   "Test Notification",
			Content:  "This is a test message",
			Priority: 3,
			Metadata: common.NotificationMetadata{
				"key1": "value1",
			},
		}

		mockService.On("SendNotification", mock.Anything, expectedEvent).Return(nil)

		resp, err := handler.SendNotification(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Equal(t, "Notification sent successfully", resp.Message)
		assert.NotEmpty(t, resp.NotificationId)

		mockService.AssertExpectations(t)
	})

	t.Run("missing required fields", func(t *testing.T) {
		handler := createTestGRPCHandler()

		testCases := []struct {
			name string
			req  *pb.SendNotificationRequest
		}{
			{
				name: "missing user_id",
				req: &pb.SendNotificationRequest{
					Title:   "Test Notification",
					Message: "This is a test message",
					Type:    "test",
				},
			},
			{
				name: "missing title",
				req: &pb.SendNotificationRequest{
					UserId:  "user123",
					Message: "This is a test message",
					Type:    "test",
				},
			},
			{
				name: "missing message",
				req: &pb.SendNotificationRequest{
					UserId: "user123",
					Title:  "Test Notification",
					Type:   "test",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				resp, err := handler.SendNotification(context.Background(), tc.req)

				assert.Error(t, err)
				assert.Nil(t, resp)

				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, codes.InvalidArgument, st.Code())
				assert.Contains(t, st.Message(), "user_id, title, and message are required")
			})
		}
	})

	t.Run("service error", func(t *testing.T) {
		mockService := &MockNotificationServiceForHandler{}
		handler := &GRPCHandler{
			service: mockService,
			config:  &config.Config{},
		}

		req := &pb.SendNotificationRequest{
			UserId:  "user123",
			Title:   "Test Notification",
			Message: "This is a test message",
			Type:    "test",
		}

		mockService.On("SendNotification", mock.Anything, mock.Anything).Return(errors.New("service error"))

		resp, err := handler.SendNotification(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "failed to send notification")

		mockService.AssertExpectations(t)
	})
}

func TestGRPCHandler_ScheduleNotification(t *testing.T) {
	t.Run("successful schedule notification", func(t *testing.T) {
		mockService := &MockNotificationServiceForHandler{}
		handler := &GRPCHandler{
			service: mockService,
			config:  &config.Config{},
		}

		futureTime := time.Now().Add(1 * time.Hour)
		req := &pb.ScheduleNotificationRequest{
			UserId:      "user123",
			Title:       "Scheduled Notification",
			Message:     "This is a scheduled message",
			Type:        "scheduled",
			ScheduledAt: timestamppb.New(futureTime),
			Data: map[string]string{
				"key1": "value1",
			},
		}

		mockService.On("ScheduleNotification", mock.Anything, mock.Anything).Return(nil)

		resp, err := handler.ScheduleNotification(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Equal(t, "Notification scheduled successfully", resp.Message)
		assert.NotEmpty(t, resp.NotificationId)

		mockService.AssertExpectations(t)
	})

	t.Run("missing scheduled_at", func(t *testing.T) {
		handler := createTestGRPCHandler()

		req := &pb.ScheduleNotificationRequest{
			UserId:  "user123",
			Title:   "Scheduled Notification",
			Message: "This is a scheduled message",
			Type:    "scheduled",
		}

		resp, err := handler.ScheduleNotification(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "user_id, title, message, and scheduled_at are required")
	})

	t.Run("scheduled_at in the past", func(t *testing.T) {
		handler := createTestGRPCHandler()

		pastTime := time.Now().Add(-1 * time.Hour)
		req := &pb.ScheduleNotificationRequest{
			UserId:      "user123",
			Title:       "Scheduled Notification",
			Message:     "This is a scheduled message",
			Type:        "scheduled",
			ScheduledAt: timestamppb.New(pastTime),
		}

		resp, err := handler.ScheduleNotification(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "scheduled_at must be in the future")
	})
}

func TestGRPCHandler_GetUserNotifications(t *testing.T) {
	t.Run("successful get notifications", func(t *testing.T) {
		mockService := &MockNotificationServiceForHandler{}
		handler := &GRPCHandler{
			service: mockService,
			config:  &config.Config{},
		}

		req := &pb.GetUserNotificationsRequest{
			UserId: "user123",
			Page:   1,
			Limit:  10,
		}

		// Create mock notification with proper metadata
		mockMetadata := &dbmysql.DBNotificationMetadata{
			"key1": "value1",
		}

		mockNotifications := []*dbmysql.Notification{
			{
				ID:        "notif1",
				UserID:    "user123",
				Type:      "test",
				Header:    "Test Notification",
				Content:   "Test content",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Metadata:  *mockMetadata,
			},
		}

		mockService.On("GetUserNotifications", mock.Anything, "user123", 10, 0).Return(mockNotifications, nil)

		resp, err := handler.GetUserNotifications(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Equal(t, "Notifications retrieved successfully", resp.Message)
		assert.Len(t, resp.Notifications, 1)
		assert.Equal(t, "notif1", resp.Notifications[0].Id)
		assert.Equal(t, int32(1), resp.TotalCount)
		assert.Equal(t, int32(1), resp.Page)
		assert.Equal(t, int32(10), resp.Limit)

		mockService.AssertExpectations(t)
	})

	t.Run("missing user_id", func(t *testing.T) {
		handler := createTestGRPCHandler()

		req := &pb.GetUserNotificationsRequest{
			Page:  1,
			Limit: 10,
		}

		resp, err := handler.GetUserNotifications(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "user_id is required")
	})

	t.Run("default pagination values", func(t *testing.T) {
		mockService := &MockNotificationServiceForHandler{}
		handler := &GRPCHandler{
			service: mockService,
			config:  &config.Config{},
		}

		req := &pb.GetUserNotificationsRequest{
			UserId: "user123",
			// Page and Limit are 0, should use defaults
		}

		mockNotifications := []*dbmysql.Notification{}
		mockService.On("GetUserNotifications", mock.Anything, "user123", 20, 0).Return(mockNotifications, nil)

		resp, err := handler.GetUserNotifications(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int32(1), resp.Page)   // Default page
		assert.Equal(t, int32(20), resp.Limit) // Default limit

		mockService.AssertExpectations(t)
	})
}

func TestGRPCHandler_MarkAsRead(t *testing.T) {
	t.Run("successful mark as read", func(t *testing.T) {
		mockService := &MockNotificationServiceForHandler{}
		handler := &GRPCHandler{
			service: mockService,
			config:  &config.Config{},
		}

		req := &pb.MarkAsReadRequest{
			NotificationId: "notif123",
			UserId:         "user123",
		}

		mockService.On("MarkAsRead", mock.Anything, "notif123", "user123").Return(nil)

		resp, err := handler.MarkAsRead(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Equal(t, "Notification marked as read", resp.Message)

		mockService.AssertExpectations(t)
	})

	t.Run("missing notification_id", func(t *testing.T) {
		handler := createTestGRPCHandler()

		req := &pb.MarkAsReadRequest{
			UserId: "user123",
		}

		resp, err := handler.MarkAsRead(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "notification_id is required")
	})
}

func TestGRPCHandler_RegisterDevice(t *testing.T) {
	t.Run("successful device registration", func(t *testing.T) {
		mockService := &MockNotificationServiceForHandler{}
		handler := &GRPCHandler{
			service: mockService,
			config:  &config.Config{},
		}

		req := &pb.RegisterDeviceRequest{
			UserId:      "user123",
			DeviceToken: "token123",
			Platform:    "ios",
		}

		mockService.On("RegisterDeviceToken", mock.Anything, "user123", "token123", "ios").Return(nil)

		resp, err := handler.RegisterDevice(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Equal(t, "Device registered successfully", resp.Message)

		mockService.AssertExpectations(t)
	})

	t.Run("missing required fields", func(t *testing.T) {
		handler := createTestGRPCHandler()

		testCases := []struct {
			name string
			req  *pb.RegisterDeviceRequest
		}{
			{
				name: "missing user_id",
				req: &pb.RegisterDeviceRequest{
					DeviceToken: "token123",
					Platform:    "ios",
				},
			},
			{
				name: "missing device_token",
				req: &pb.RegisterDeviceRequest{
					UserId:   "user123",
					Platform: "ios",
				},
			},
			{
				name: "missing platform",
				req: &pb.RegisterDeviceRequest{
					UserId:      "user123",
					DeviceToken: "token123",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				resp, err := handler.RegisterDevice(context.Background(), tc.req)

				assert.Error(t, err)
				assert.Nil(t, resp)

				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, codes.InvalidArgument, st.Code())
				assert.Contains(t, st.Message(), "user_id, device_token, and platform are required")
			})
		}
	})
}

func TestGRPCHandler_SendFriendRequest(t *testing.T) {
	t.Run("successful friend request", func(t *testing.T) {
		mockService := &MockNotificationServiceForHandler{}
		handler := &GRPCHandler{
			service: mockService,
			config:  &config.Config{},
		}

		req := &pb.SendFriendRequestRequest{
			FromUserId:   "user123",
			ToUserId:     "user456",
			FromUsername: "john_doe",
		}

		mockService.On("SendFriendRequestNotification", mock.Anything, "user123", "user456", "john_doe").Return(nil)

		resp, err := handler.SendFriendRequest(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Equal(t, "Friend request notification sent successfully", resp.Message)
		assert.NotEmpty(t, resp.NotificationId)

		mockService.AssertExpectations(t)
	})

	t.Run("missing required fields", func(t *testing.T) {
		handler := createTestGRPCHandler()

		req := &pb.SendFriendRequestRequest{
			FromUserId: "user123",
			// Missing ToUserId and FromUsername
		}

		resp, err := handler.SendFriendRequest(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "from_user_id, to_user_id, and from_username are required")
	})
}

func TestGRPCHandler_HealthCheck(t *testing.T) {
	t.Run("health check", func(t *testing.T) {
		handler := createTestGRPCHandler()

		req := &pb.HealthCheckRequest{}

		resp, err := handler.HealthCheck(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "healthy", resp.Status)
		assert.Equal(t, "gosocial-notifications-grpc", resp.Service)
		assert.NotNil(t, resp.Timestamp)
	})
}

// Test helper functions
func TestConvertMapToMetadata(t *testing.T) {
	input := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	result := convertMapToMetadata(input)

	assert.Len(t, result, 2)
	assert.Equal(t, "value1", result["key1"])
	assert.Equal(t, "value2", result["key2"])
}

func TestConvertDBNotificationMetadataToMap(t *testing.T) {
	input := dbmysql.DBNotificationMetadata{
		"key1": "value1",
		"key2": "value2",
		"key3": 123, // Non-string value should be ignored
	}

	result := convertDBNotificationMetadataToMap(input)

	assert.Len(t, result, 2) // Only string values should be included
	assert.Equal(t, "value1", result["key1"])
	assert.Equal(t, "value2", result["key2"])
	assert.NotContains(t, result, "key3")
}

func TestGenerateNotificationID(t *testing.T) {
	id1 := generateNotificationID()
	id2 := generateNotificationID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "notif_")
	assert.Contains(t, id2, "notif_")
}

// Benchmark tests
func BenchmarkGRPCHandler_SendNotification(b *testing.B) {
	mockService := &MockNotificationServiceForHandler{}
	handler := &GRPCHandler{
		service: mockService,
		config:  &config.Config{},
	}

	req := &pb.SendNotificationRequest{
		UserId:  "user123",
		Title:   "Benchmark Test",
		Message: "Benchmark message",
		Type:    "test",
	}

	mockService.On("SendNotification", mock.Anything, mock.Anything).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.SendNotification(context.Background(), req)
	}
}

func BenchmarkGRPCHandler_GetUserNotifications(b *testing.B) {
	mockService := &MockNotificationServiceForHandler{}
	handler := &GRPCHandler{
		service: mockService,
		config:  &config.Config{},
	}

	req := &pb.GetUserNotificationsRequest{
		UserId: "user123",
		Page:   1,
		Limit:  20,
	}

	mockNotifications := []*dbmysql.Notification{}
	mockService.On("GetUserNotifications", mock.Anything, "user123", 20, 0).Return(mockNotifications, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.GetUserNotifications(context.Background(), req)
	}
}

// Integration tests
func TestGRPCHandler_Integration(t *testing.T) {
	t.Run("full notification workflow", func(t *testing.T) {
		mockService := &MockNotificationServiceForHandler{}
		handler := &GRPCHandler{
			service: mockService,
			config:  &config.Config{},
		}

		// Register device
		mockService.On("RegisterDeviceToken", mock.Anything, "user123", "token123", "ios").Return(nil)

		registerReq := &pb.RegisterDeviceRequest{
			UserId:      "user123",
			DeviceToken: "token123",
			Platform:    "ios",
		}

		registerResp, err := handler.RegisterDevice(context.Background(), registerReq)
		assert.NoError(t, err)
		assert.True(t, registerResp.Success)

		// Send notification
		mockService.On("SendNotification", mock.Anything, mock.Anything).Return(nil)

		sendReq := &pb.SendNotificationRequest{
			UserId:  "user123",
			Title:   "Integration Test",
			Message: "This is an integration test",
			Type:    "test",
		}

		sendResp, err := handler.SendNotification(context.Background(), sendReq)
		assert.NoError(t, err)
		assert.True(t, sendResp.Success)

		mockService.AssertExpectations(t)
	})
}
