package notif

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "gosocial/api/v1"
	"gosocial/internal/common"
	"gosocial/internal/config"
	"gosocial/internal/dbmysql"

	"firebase.google.com/go/v4/messaging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NotificationHandler wraps the gRPC handler
type NotificationHandler struct {
	GRPC *GRPCHandler
}

// GRPCHandler implements the notification gRPC service
type GRPCHandler struct {
	pb.UnimplementedNotificationServiceServer
	service    *NotificationService
	config     *config.Config
	fcmClient  *messaging.Client
	deviceRepo common.DeviceRepository
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(
	service *NotificationService,
	config *config.Config,
	fcmClient *messaging.Client,
	deviceRepo common.DeviceRepository,
) *NotificationHandler {
	grpcHandler := NewGRPCHandler(service, config, fcmClient, deviceRepo)
	return &NotificationHandler{
		GRPC: grpcHandler,
	}
}

// NewGRPCHandler creates a new gRPC handler
func NewGRPCHandler(
	service *NotificationService,
	config *config.Config,
	fcmClient *messaging.Client,
	deviceRepo common.DeviceRepository,
) *GRPCHandler {
	return &GRPCHandler{
		service:    service,
		config:     config,
		fcmClient:  fcmClient,
		deviceRepo: deviceRepo,
	}
}

// SendNotification sends an immediate notification
func (h *GRPCHandler) SendNotification(ctx context.Context, req *pb.SendNotificationRequest) (*pb.SendNotificationResponse, error) {
	log.Printf("gRPC SendNotification called for user: %s", req.UserId)

	if req.UserId == "" || req.Title == "" || req.Message == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, title, and message are required")
	}

	// Create notification event
	event := common.NotificationEvent{
		Type:     common.NotificationType(req.Type),
		UserID:   req.UserId,
		Header:   req.Title,
		Content:  req.Message,
		Priority: 3, // Default priority
		Metadata: convertMapToMetadata(req.Data),
	}

	// Send notification through service
	err := h.service.SendNotification(ctx, event)
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

// ScheduleNotification schedules a notification for later delivery
func (h *GRPCHandler) ScheduleNotification(ctx context.Context, req *pb.ScheduleNotificationRequest) (*pb.ScheduleNotificationResponse, error) {
	log.Printf("gRPC ScheduleNotification called for user: %s", req.UserId)

	if req.UserId == "" || req.Title == "" || req.Message == "" || req.ScheduledAt == nil {
		return nil, status.Error(codes.InvalidArgument, "user_id, title, message, and scheduled_at are required")
	}

	scheduledTime := req.ScheduledAt.AsTime()
	if scheduledTime.Before(time.Now()) {
		return nil, status.Error(codes.InvalidArgument, "scheduled_at must be in the future")
	}

	// Create notification event
	event := common.NotificationEvent{
		Type:        common.NotificationType(req.Type),
		UserID:      req.UserId,
		Header:      req.Title,
		Content:     req.Message,
		Priority:    2,
		ScheduledAt: &scheduledTime,
		Metadata:    convertMapToMetadata(req.Data),
	}

	// Schedule notification through service
	err := h.service.ScheduleNotification(ctx, event)
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

// GetUserNotifications retrieves notifications for a user
func (h *GRPCHandler) GetUserNotifications(ctx context.Context, req *pb.GetUserNotificationsRequest) (*pb.GetUserNotificationsResponse, error) {
	log.Printf("gRPC GetUserNotifications called for user: %s", req.UserId)

	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// Set default pagination
	page := req.Page
	if page <= 0 {
		page = 1
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}

	// Calculate offset
	offset := int((page - 1) * limit)

	// Get notifications from service
	notifications, err := h.service.GetUserNotifications(ctx, req.UserId, int(limit), offset)
	if err != nil {
		log.Printf("Failed to get user notifications: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get notifications: %v", err))
	}

	// Convert to proto format
	pbNotifications := make([]*pb.NotificationData, len(notifications))
	for i, notif := range notifications {
		pbNotifications[i] = &pb.NotificationData{
			Id:        notif.ID,
			UserId:    req.UserId,
			Title:     notif.Header,
			Message:   notif.Content,
			Type:      string(notif.Type),
			IsRead:    notif.ReadAt != nil,
			CreatedAt: timestamppb.New(notif.CreatedAt),
			UpdatedAt: timestamppb.New(notif.UpdatedAt),
			Data:      convertDBNotificationMetadataToMap(notif.Metadata), // Fixed function call
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

// MarkAsRead marks a notification as read
func (h *GRPCHandler) MarkAsRead(ctx context.Context, req *pb.MarkAsReadRequest) (*pb.MarkAsReadResponse, error) {
	log.Printf("gRPC MarkAsRead called for notification: %s", req.NotificationId)

	if req.NotificationId == "" {
		return nil, status.Error(codes.InvalidArgument, "notification_id is required")
	}

	// Mark as read through service
	err := h.service.MarkAsRead(ctx, req.NotificationId, req.UserId)
	if err != nil {
		log.Printf("Failed to mark notification as read: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to mark as read: %v", err))
	}

	return &pb.MarkAsReadResponse{
		Success: true,
		Message: "Notification marked as read",
	}, nil
}

// RegisterDevice registers a device for push notifications
func (h *GRPCHandler) RegisterDevice(ctx context.Context, req *pb.RegisterDeviceRequest) (*pb.RegisterDeviceResponse, error) {
	log.Printf("gRPC RegisterDevice called for user: %s", req.UserId)

	if req.UserId == "" || req.DeviceToken == "" || req.Platform == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, device_token, and platform are required")
	}

	// Register device through service
	err := h.service.RegisterDeviceToken(ctx, req.UserId, req.DeviceToken, req.Platform)
	if err != nil {
		log.Printf("Failed to register device: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to register device: %v", err))
	}

	return &pb.RegisterDeviceResponse{
		Success: true,
		Message: "Device registered successfully",
	}, nil
}

// SendFriendRequest sends a friend request notification
func (h *GRPCHandler) SendFriendRequest(ctx context.Context, req *pb.SendFriendRequestRequest) (*pb.SendFriendRequestResponse, error) {
	log.Printf("gRPC SendFriendRequest called from %s to %s", req.FromUserId, req.ToUserId)

	if req.FromUserId == "" || req.ToUserId == "" || req.FromUsername == "" {
		return nil, status.Error(codes.InvalidArgument, "from_user_id, to_user_id, and from_username are required")
	}

	// Send friend request notification through service
	err := h.service.SendFriendRequestNotification(ctx, req.FromUserId, req.ToUserId, req.FromUsername)
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

// HealthCheck returns the health status of the service
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

// Fixed: Create separate function for converting DBNotificationMetadata to map
func convertDBNotificationMetadataToMap(metadata dbmysql.DBNotificationMetadata) map[string]string {
	result := make(map[string]string)
	for k, v := range metadata {
		if str, ok := v.(string); ok {
			result[k] = str
		}
	}
	return result
}

// Keep the original function for common.NotificationMetadata
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
	return fmt.Sprintf("notif_%d", time.Now().UnixNano())
}

func generateID() string {
	return fmt.Sprintf("notif_%d", time.Now().UnixNano())
}

func convertMetadataToMap(metadata dbmysql.DBNotificationMetadata) map[string]string {
	result := make(map[string]string)
	for k, v := range metadata {
		if str, ok := v.(string); ok {
			result[k] = str
		}
	}
	return result
}
