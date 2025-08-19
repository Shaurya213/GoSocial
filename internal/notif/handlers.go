package notif

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	pb "gosocial/api/v1"
	"gosocial/internal/common"
	"gosocial/internal/config"
	"gosocial/internal/dbmysql"
	"gosocial/internal/user"

	"firebase.google.com/go/v4/messaging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type NotificationServiceInterface interface {
	SendNotification(ctx context.Context, event common.NotificationEvent) error
	ScheduleNotification(ctx context.Context, event common.NotificationEvent) error

	GetUserNotifications(ctx context.Context, userID uint64, limit, offset int) ([]*dbmysql.Notification, error)
	MarkAsRead(ctx context.Context, notificationID, userID uint) error
	RegisterDeviceToken(ctx context.Context, userID uint, deviceToken, platform string) error
	SendFriendRequestNotification(ctx context.Context, fromUserID uint, toUserID uint, fromUsername string) error
	Shutdown()
}

type NotificationHandler struct {
	GRPC *GRPCHandler
}

type GRPCHandler struct {
	pb.UnimplementedNotificationServiceServer
	service    NotificationServiceInterface // interface instead of concrete type
	config     *config.Config
	fcmClient  *messaging.Client
	deviceRepo user.DeviceRepository
}

func NewNotificationHandler(
	service NotificationServiceInterface, // Changed to interface
	config *config.Config,
	fcmClient *messaging.Client,
	deviceRepo user.DeviceRepository,

) *NotificationHandler {
	grpcHandler := NewGRPCHandler(service, config, fcmClient, deviceRepo)
	return &NotificationHandler{
		GRPC: grpcHandler,
	}
}

func NewGRPCHandler(
	service NotificationServiceInterface, // Changed to interface
	config *config.Config,
	fcmClient *messaging.Client,
	deviceRepo user.DeviceRepository,

) *GRPCHandler {
	return &GRPCHandler{
		service:    service,
		config:     config,
		fcmClient:  fcmClient,
		deviceRepo: deviceRepo,
	}
}

// Send notification immediately
func (h *GRPCHandler) SendNotification(ctx context.Context, req *pb.SendNotificationRequest) (*pb.SendNotificationResponse, error) {
	log.Printf("gRPC SendNotification called for user: %s", req.UserId)
	if req.UserId == "" || req.Title == "" || req.Message == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, title, and message are required")
	}

	// Convert proto request to internal domain model

	var userIDUint uint
	_, err := fmt.Sscanf(req.UserId, "%d", &userIDUint)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "user_id must be a valid unsigned integer")
	}
	event := common.NotificationEvent{

		Type:     common.NotificationType(req.Type),
		UserID:   userIDUint,
		Header:   req.Title,
		Content:  req.Message,
		Priority: 3, // Default priority
		Metadata: convertMapToMetadata(req.Data),
	}

	err = h.service.SendNotification(ctx, event)

	if err != nil {
		log.Printf("Failed to send notification: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to send notification: %v", err))
	}

	return &pb.SendNotificationResponse{
		Success:        true,
		Message:        "Notification sent successfully",
		NotificationId: generateNotificationID(),
	}, nil
}

// Schedule notification for later delivery
func (h *GRPCHandler) ScheduleNotification(ctx context.Context, req *pb.ScheduleNotificationRequest) (*pb.ScheduleNotificationResponse, error) {
	log.Printf("gRPC ScheduleNotification called for user: %s", req.UserId)
	if req.UserId == "" || req.Title == "" || req.Message == "" || req.ScheduledAt == nil {
		return nil, status.Error(codes.InvalidArgument, "user_id, title, message, and scheduled_at are required")
	}

	scheduledTime := req.ScheduledAt.AsTime()
	if scheduledTime.Before(time.Now()) {
		return nil, status.Error(codes.InvalidArgument, "scheduled_at must be in the future")
	}

	var userIDUint uint
	_, err := fmt.Sscanf(req.UserId, "%d", &userIDUint)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "user_id must be a valid unsigned integer")
	}
	event := common.NotificationEvent{

		Type:        common.NotificationType(req.Type),
		UserID:      userIDUint,
		Header:      req.Title,
		Content:     req.Message,
		Priority:    2,
		ScheduledAt: &scheduledTime,
		Metadata:    convertMapToMetadata(req.Data),
	}

	err = h.service.ScheduleNotification(ctx, event)

	if err != nil {
		log.Printf("Failed to schedule notification: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to schedule notification: %v", err))
	}

	return &pb.ScheduleNotificationResponse{
		Success:        true,
		Message:        "Notification scheduled successfully",
		NotificationId: generateNotificationID(),
	}, nil
}

// Get notifications for a user
func (h *GRPCHandler) GetUserNotifications(ctx context.Context, req *pb.GetUserNotificationsRequest) (*pb.GetUserNotificationsResponse, error) {
	log.Printf("gRPC GetUserNotifications called for user: %s", req.UserId)
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	page := req.Page //default pagination
	if page <= 0 {
		page = 1
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}

	// Calculate offset
	offset := int((page - 1) * limit)

	var userIDUint64 uint64
	_, err := fmt.Sscanf(req.UserId, "%d", &userIDUint64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "user_id must be a valid unsigned integer")
	}

	notifications, err := h.service.GetUserNotifications(ctx, userIDUint64, int(limit), offset)

	if err != nil {
		log.Printf("Failed to get user notifications: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get notifications: %v", err))
	}

	// Converting format to proto format
	pbNotifications := make([]*pb.NotificationData, len(notifications))
	for i, notif := range notifications {
		pbNotifications[i] = &pb.NotificationData{

			Id:     fmt.Sprintf("%d", notif.ID),
			UserId: fmt.Sprintf("%d", userIDUint64),

			Title:     notif.Header,
			Message:   notif.Content,
			Type:      string(notif.Type),
			IsRead:    notif.ReadAt != nil,
			CreatedAt: timestamppb.New(notif.CreatedAt),
			UpdatedAt: timestamppb.New(notif.UpdatedAt),
			Data:      convertDBNotificationMetadataToMap(notif.Metadata),
		}
	}

	return &pb.GetUserNotificationsResponse{
		Success:       true,
		Message:       "Notifications retrieved successfully",
		Notifications: pbNotifications,
		TotalCount:    int32(len(notifications)),
		Page:          page,
		Limit:         limit,
	}, nil
}

// Mark notification as read
func (h *GRPCHandler) MarkAsRead(ctx context.Context, req *pb.MarkAsReadRequest) (*pb.MarkAsReadResponse, error) {
	log.Printf("gRPC MarkAsRead called for notification: %s", req.NotificationId)
	if req.NotificationId == "" {
		return nil, status.Error(codes.InvalidArgument, "notification_id is required")
	}

	var notificationIDUint uint
	_, err := fmt.Sscanf(req.NotificationId, "%d", &notificationIDUint)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "notification_id must be a valid unsigned integer")
	}

	var userIDUint uint
	_, err = fmt.Sscanf(req.UserId, "%d", &userIDUint)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "user_id must be a valid unsigned integer")
	}

	err = h.service.MarkAsRead(ctx, notificationIDUint, userIDUint)

	if err != nil {
		log.Printf("Failed to mark notification as read: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to mark as read: %v", err))
	}

	return &pb.MarkAsReadResponse{
		Success: true,
		Message: "Notification marked as read",
	}, nil
}

// Register device for push notifications
func (h *GRPCHandler) RegisterDevice(ctx context.Context, req *pb.RegisterDeviceRequest) (*pb.RegisterDeviceResponse, error) {
	log.Printf("gRPC RegisterDevice called for user: %s", req.UserId)
	if req.UserId == "" || req.DeviceToken == "" || req.Platform == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, device_token, and platform are required")
	}

	var userIDUint uint
	_, err := fmt.Sscanf(req.UserId, "%d", &userIDUint)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "user_id must be a valid unsigned integer")
	}

	err = h.service.RegisterDeviceToken(ctx, userIDUint, req.DeviceToken, req.Platform)

	if err != nil {
		log.Printf("Failed to register device: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to register device: %v", err))
	}

	return &pb.RegisterDeviceResponse{
		Success: true,
		Message: "Device registered successfully",
	}, nil
}

// Send friend request notification
func (h *GRPCHandler) SendFriendRequest(ctx context.Context, req *pb.SendFriendRequestRequest) (*pb.SendFriendRequestResponse, error) {
	log.Printf("gRPC SendFriendRequest called from %s to %s", req.FromUserId, req.ToUserId)
	if req.FromUserId == "" || req.ToUserId == "" || req.FromUsername == "" {
		return nil, status.Error(codes.InvalidArgument, "from_user_id, to_user_id, and from_username are required")
	}

	var fromUserIDUint, toUserIDUint uint
	_, err := fmt.Sscanf(req.FromUserId, "%d", &fromUserIDUint)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "from_user_id must be a valid unsigned integer")
	}
	_, err = fmt.Sscanf(req.ToUserId, "%d", &toUserIDUint)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "to_user_id must be a valid unsigned integer")
	}

	err = h.service.SendFriendRequestNotification(ctx, fromUserIDUint, toUserIDUint, req.FromUsername)

	if err != nil {
		log.Printf("Failed to send friend request notification: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to send friend request: %v", err))
	}

	return &pb.SendFriendRequestResponse{
		Success:        true,
		Message:        "Friend request notification sent successfully",
		NotificationId: generateNotificationID(),
	}, nil
}

// Health check
func (h *GRPCHandler) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Status:    "healthy",
		Service:   "gosocial-notifications-grpc",
		Timestamp: timestamppb.New(time.Now()),
	}, nil
}

// Helper functions
func convertMapToMetadata(data map[string]string) common.NotificationMetadata {
	metadata := make(common.NotificationMetadata)
	for k, v := range data {
		metadata[k] = v
	}
	return metadata
}

func convertDBNotificationMetadataToMap(metadata dbmysql.DBNotificationMetadata) map[string]string {
	result := make(map[string]string)
	for k, v := range metadata {
		if str, ok := v.(string); ok {
			result[k] = str
		}
	}
	return result
}

func convertNotificationMetadataToMap(metadata common.NotificationMetadata) map[string]string {
	result := make(map[string]string)
	for k, v := range metadata {
		if str, ok := v.(string); ok {
			result[k] = str
		}
	}
	return result
}

func generateNotificationID() string {

	timestamp := time.Now().UnixNano()

	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomStr := hex.EncodeToString(randomBytes)

	return fmt.Sprintf("notif_%d_%s", timestamp, randomStr)
}
