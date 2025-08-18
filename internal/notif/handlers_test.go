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

// --- Mock Services ---

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

func (m *MockNotificationServiceForHandler) GetUserNotifications(ctx context.Context, userID uint64, limit, offset int) ([]*dbmysql.Notification, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]*dbmysql.Notification), args.Error(1)
}

func (m *MockNotificationServiceForHandler) MarkAsRead(ctx context.Context, notificationID, userID uint) error {
	args := m.Called(ctx, notificationID, userID)
	return args.Error(0)
}

func (m *MockNotificationServiceForHandler) RegisterDeviceToken(ctx context.Context, userID uint, deviceToken, platform string) error {
	args := m.Called(ctx, userID, deviceToken, platform)
	return args.Error(0)
}

func (m *MockNotificationServiceForHandler) SendFriendRequestNotification(ctx context.Context, fromUserID, toUserID uint, fromUsername string) error {
	args := m.Called(ctx, fromUserID, toUserID, fromUsername)
	return args.Error(0)
}

func (m *MockNotificationServiceForHandler) Shutdown() {
	m.Called()
}

type MockDeviceRepositoryForHandler struct {
	mock.Mock
}

func (m *MockDeviceRepositoryForHandler) GetUserDevices(ctx context.Context, userID uint64) ([]*dbmysql.Device, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*dbmysql.Device), args.Error(1)
}

func (m *MockDeviceRepositoryForHandler) CreateOrUpdate(ctx context.Context, userID uint64, deviceToken, platform string) error {
	args := m.Called(ctx, userID, deviceToken, platform)
	return args.Error(0)
}

func (m *MockDeviceRepositoryForHandler) ActiveByUserID(ctx context.Context, userID uint64) ([]interface{}, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *MockDeviceRepositoryForHandler) RegisterDevice(ctx context.Context, device *dbmysql.Device) error {
	args := m.Called(ctx, device)
	return args.Error(0)
}

func (m *MockDeviceRepositoryForHandler) UpdateTokenStatus(ctx context.Context, token string, isActive bool) error {
	args := m.Called(ctx, token, isActive)
	return args.Error(0)
}

func (m *MockDeviceRepositoryForHandler) UpdatedDeviceActivity(ctx context.Context, deviceToken string) error {
	args := m.Called(ctx, deviceToken)
	return args.Error(0)
}

func (m *MockDeviceRepositoryForHandler) DeleteToken(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockDeviceRepositoryForHandler) RemovedDevice(ctx context.Context, deviceToken string) error {
	args := m.Called(ctx, deviceToken)
	return args.Error(0)
}

// --- Setup Helper ---

func createTestGRPCHandler() *GRPCHandler {
	mockService := &MockNotificationServiceForHandler{}
	mockConfig := &config.Config{}
	mockDeviceRepo := &MockDeviceRepositoryForHandler{}
	return NewGRPCHandler(mockService, mockConfig, nil, mockDeviceRepo)
}

// --- Test New Handlers ---

func TestNewNotificationHandler(t *testing.T) {
	mockService := &MockNotificationServiceForHandler{}
	mockConfig := &config.Config{}
	mockDeviceRepo := &MockDeviceRepositoryForHandler{}
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

// --- SendNotification ---

func TestGRPCHandler_SendNotification(t *testing.T) {
	t.Run("successful send notification", func(t *testing.T) {
		mockService := &MockNotificationServiceForHandler{}
		handler := &GRPCHandler{service: mockService, config: &config.Config{}}
		req := &pb.SendNotificationRequest{
			UserId:  "123",
			Title:   "Test Notification",
			Message: "This is a test message",
			Type:    "test",
			Data:    map[string]string{"k": "v"},
		}
		expectedEvent := common.NotificationEvent{
			Type:     common.NotificationType("test"),
			UserID:   uint(123),
			Header:   "Test Notification",
			Content:  "This is a test message",
			Priority: 3,
			Metadata: common.NotificationMetadata{"k": "v"},
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
			{"missing user_id", &pb.SendNotificationRequest{Title: "t", Message: "m", Type: "type"}},
			{"missing title", &pb.SendNotificationRequest{UserId: "123", Message: "m", Type: "type"}},
			{"missing message", &pb.SendNotificationRequest{UserId: "123", Title: "t", Type: "type"}},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				resp, err := handler.SendNotification(context.Background(), tc.req)
				assert.Error(t, err)
				assert.Nil(t, resp)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, codes.InvalidArgument, st.Code())
			})
		}
	})

	t.Run("invalid user_id type", func(t *testing.T) {
		handler := createTestGRPCHandler()
		req := &pb.SendNotificationRequest{
			UserId:  "notanumber",
			Title:   "Test",
			Message: "Test",
			Type:    "test",
		}
		resp, err := handler.SendNotification(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("error from service", func(t *testing.T) {
		mockService := &MockNotificationServiceForHandler{}
		handler := &GRPCHandler{service: mockService, config: &config.Config{}}
		req := &pb.SendNotificationRequest{
			UserId:  "123",
			Title:   "Error",
			Message: "Error",
			Type:    "err",
		}
		mockService.On("SendNotification", mock.Anything, mock.Anything).Return(errors.New("service error"))
		resp, err := handler.SendNotification(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Internal, st.Code())
		mockService.AssertExpectations(t)
	})
}

// --- ScheduleNotification ---

func TestGRPCHandler_ScheduleNotification(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockService := &MockNotificationServiceForHandler{}
		handler := &GRPCHandler{service: mockService, config: &config.Config{}}
		sched := time.Now().Add(time.Hour)
		req := &pb.ScheduleNotificationRequest{
			UserId:      "123",
			Title:       "Sched",
			Message:     "scheduled!",
			Type:        "scheduled",
			ScheduledAt: timestamppb.New(sched),
			Data:        map[string]string{"xx": "yy"},
		}
		mockService.On("ScheduleNotification", mock.Anything, mock.Anything).Return(nil)
		resp, err := handler.ScheduleNotification(context.Background(), req)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.NotificationId)
		mockService.AssertExpectations(t)
	})

	t.Run("missing scheduled_at", func(t *testing.T) {
		handler := createTestGRPCHandler()
		req := &pb.ScheduleNotificationRequest{
			UserId: "123", Title: "t", Message: "m", Type: "scheduled",
		}
		resp, err := handler.ScheduleNotification(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("scheduled_at in the past", func(t *testing.T) {
		handler := createTestGRPCHandler()
		req := &pb.ScheduleNotificationRequest{
			UserId:      "123",
			Title:       "Past",
			Message:     "Past notification",
			Type:        "scheduled",
			ScheduledAt: timestamppb.New(time.Now().Add(-2 * time.Hour)),
		}
		resp, err := handler.ScheduleNotification(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("invalid user_id", func(t *testing.T) {
		handler := createTestGRPCHandler()
		req := &pb.ScheduleNotificationRequest{
			UserId:      "aa",
			Title:       "t",
			Message:     "m",
			Type:        "scheduled",
			ScheduledAt: timestamppb.New(time.Now().Add(time.Hour)),
		}
		resp, err := handler.ScheduleNotification(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

// --- GetUserNotifications ---

func TestGRPCHandler_GetUserNotifications(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockService := &MockNotificationServiceForHandler{}
		handler := &GRPCHandler{service: mockService, config: &config.Config{}}
		req := &pb.GetUserNotificationsRequest{UserId: "123", Page: 1, Limit: 10}
		metadata := dbmysql.DBNotificationMetadata{"key1": "value1"}
		mockNotifications := []*dbmysql.Notification{
			{
				ID:        1,
				UserID:    123,
				Type:      "test",
				Header:    "header",
				Content:   "content",
				Metadata:  metadata,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		mockService.On("GetUserNotifications", mock.Anything, uint64(123), 10, 0).Return(mockNotifications, nil)
		resp, err := handler.GetUserNotifications(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.Len(t, resp.Notifications, 1)
		mockService.AssertExpectations(t)
	})

	t.Run("missing user_id", func(t *testing.T) {
		handler := createTestGRPCHandler()
		req := &pb.GetUserNotificationsRequest{}
		resp, err := handler.GetUserNotifications(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("invalid user_id", func(t *testing.T) {
		handler := createTestGRPCHandler()
		req := &pb.GetUserNotificationsRequest{UserId: "a"}
		resp, err := handler.GetUserNotifications(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("default pagination", func(t *testing.T) {
		mockService := &MockNotificationServiceForHandler{}
		handler := &GRPCHandler{service: mockService, config: &config.Config{}}
		req := &pb.GetUserNotificationsRequest{UserId: "123"}
		mockService.On("GetUserNotifications", mock.Anything, uint64(123), 20, 0).Return([]*dbmysql.Notification{}, nil)
		resp, err := handler.GetUserNotifications(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), resp.Page)
		assert.Equal(t, int32(20), resp.Limit)
		mockService.AssertExpectations(t)
	})
}

// --- MarkAsRead ---

func TestGRPCHandler_MarkAsRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockService := &MockNotificationServiceForHandler{}
		handler := &GRPCHandler{service: mockService, config: &config.Config{}}
		req := &pb.MarkAsReadRequest{NotificationId: "123", UserId: "123"}
		mockService.On("MarkAsRead", mock.Anything, uint(123), uint(123)).Return(nil)
		resp, err := handler.MarkAsRead(context.Background(), req)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		mockService.AssertExpectations(t)
	})

	t.Run("missing notification_id", func(t *testing.T) {
		handler := createTestGRPCHandler()
		req := &pb.MarkAsReadRequest{UserId: "123"}
		resp, err := handler.MarkAsRead(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("invalid notification_id", func(t *testing.T) {
		handler := createTestGRPCHandler()
		req := &pb.MarkAsReadRequest{NotificationId: "blah", UserId: "123"}
		resp, err := handler.MarkAsRead(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("invalid user_id", func(t *testing.T) {
		handler := createTestGRPCHandler()
		req := &pb.MarkAsReadRequest{NotificationId: "123", UserId: "b"}
		resp, err := handler.MarkAsRead(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

// --- RegisterDevice ---

func TestGRPCHandler_RegisterDevice(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockService := &MockNotificationServiceForHandler{}
		handler := &GRPCHandler{service: mockService, config: &config.Config{}}
		req := &pb.RegisterDeviceRequest{
			UserId:      "123",
			DeviceToken: "token123",
			Platform:    "ios",
		}
		mockService.On("RegisterDeviceToken", mock.Anything, uint(123), "token123", "ios").Return(nil)
		resp, err := handler.RegisterDevice(context.Background(), req)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		mockService.AssertExpectations(t)
	})

	t.Run("missing required fields", func(t *testing.T) {
		handler := createTestGRPCHandler()
		testCases := []struct {
			name string
			req  *pb.RegisterDeviceRequest
		}{
			{"missing user_id", &pb.RegisterDeviceRequest{DeviceToken: "tok", Platform: "i"}},
			{"missing device_token", &pb.RegisterDeviceRequest{UserId: "123", Platform: "i"}},
			{"missing platform", &pb.RegisterDeviceRequest{UserId: "123", DeviceToken: "t"}},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				resp, err := handler.RegisterDevice(context.Background(), tc.req)
				assert.Error(t, err)
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, codes.InvalidArgument, st.Code())
			})
		}
	})

	t.Run("invalid user_id", func(t *testing.T) {
		handler := createTestGRPCHandler()
		req := &pb.RegisterDeviceRequest{UserId: "abc", DeviceToken: "t", Platform: "i"}
		resp, err := handler.RegisterDevice(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

// --- SendFriendRequest ---

func TestGRPCHandler_SendFriendRequest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockService := &MockNotificationServiceForHandler{}
		handler := &GRPCHandler{service: mockService, config: &config.Config{}}
		req := &pb.SendFriendRequestRequest{
			FromUserId:   "123",
			ToUserId:     "456",
			FromUsername: "john_doe",
		}
		mockService.On("SendFriendRequestNotification", mock.Anything, uint(123), uint(456), "john_doe").Return(nil)
		resp, err := handler.SendFriendRequest(context.Background(), req)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		mockService.AssertExpectations(t)
	})

	t.Run("missing fields", func(t *testing.T) {
		handler := createTestGRPCHandler()
		req := &pb.SendFriendRequestRequest{FromUserId: "123"}
		resp, err := handler.SendFriendRequest(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("invalid from_user_id", func(t *testing.T) {
		handler := createTestGRPCHandler()
		req := &pb.SendFriendRequestRequest{FromUserId: "bad", ToUserId: "456", FromUsername: "john"}
		resp, err := handler.SendFriendRequest(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("invalid to_user_id", func(t *testing.T) {
		handler := createTestGRPCHandler()
		req := &pb.SendFriendRequestRequest{FromUserId: "123", ToUserId: "bad", FromUsername: "john"}
		resp, err := handler.SendFriendRequest(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

// --- HealthCheck ---

func TestGRPCHandler_HealthCheck(t *testing.T) {
	handler := createTestGRPCHandler()
	req := &pb.HealthCheckRequest{}
	resp, err := handler.HealthCheck(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "healthy", resp.Status)
	assert.Equal(t, "gosocial-notifications-grpc", resp.Service)
	assert.NotNil(t, resp.Timestamp)
}

// --- Helper Functions ---

func TestConvertMapToMetadata(t *testing.T) {
	input := map[string]string{"k": "v"}
	result := convertMapToMetadata(input)
	assert.Len(t, result, 1)
	assert.Equal(t, "v", result["k"])
}

func TestConvertDBNotificationMetadataToMap(t *testing.T) {
	input := dbmysql.DBNotificationMetadata{
		"a": "b",
		"x": 123, // should ignore int
	}
	result := convertDBNotificationMetadataToMap(input)
	assert.Len(t, result, 1)
	assert.Equal(t, "b", result["a"])
	assert.NotContains(t, result, "x")
}

func TestGenerateNotificationID(t *testing.T) {
	id1 := generateNotificationID()
	id2 := generateNotificationID()
	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "notif_")
}
