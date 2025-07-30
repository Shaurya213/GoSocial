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

type NotificationHandler struct { //distinguieshes it with other handlers and wraps it
	GRPC *GRPCHandler
}

type GRPCHandler struct { //handler services of grpc handler
	pb.UnimplementedNotificationServiceServer
	service    *NotificationService    //business logic
	config     *config.Config          //applicatio configuration
	fcmClient  *messaging.Client       //firebase confirguration which is disabled right now
	deviceRepo common.DeviceRepository //device token storage
}

// basically a constructor for Notification handler
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

// Factory patter , helps in dependencies injection, encapsulates handler creation and dependencies injection
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

// imdeiately sends notification
func (h *GRPCHandler) SendNotification(ctx context.Context, req *pb.SendNotificationRequest) (*pb.SendNotificationResponse, error) {
	log.Printf("gRPC SendNotification called for user: %s", req.UserId)

	if req.UserId == "" || req.Title == "" || req.Message == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, title, and message are required") //improves that no invalid entries are added in the server
	}

	// convert proto  request to internal domain model and also helps in type conversion
	event := common.NotificationEvent{
		Type:     common.NotificationType(req.Type),
		UserID:   req.UserId,
		Header:   req.Title,
		Content:  req.Message,
		Priority: 3, // Default priority
		Metadata: convertMapToMetadata(req.Data),
	}

	// returns succes response with generated id
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

// ScheduleNotification  for later delivery
func (h *GRPCHandler) ScheduleNotification(ctx context.Context, req *pb.ScheduleNotificationRequest) (*pb.ScheduleNotificationResponse, error) {
	log.Printf("gRPC ScheduleNotification called for user: %s", req.UserId)

	if req.UserId == "" || req.Title == "" || req.Message == "" || req.ScheduledAt == nil {
		return nil, status.Error(codes.InvalidArgument, "user_id, title, message, and scheduled_at are required")
	}

	scheduledTime := req.ScheduledAt.AsTime()
	if scheduledTime.Before(time.Now()) {
		return nil, status.Error(codes.InvalidArgument, "scheduled_at must be in the future")
	}

	// proto conversion and type casting
	event := common.NotificationEvent{
		Type:        common.NotificationType(req.Type),
		UserID:      req.UserId,
		Header:      req.Title,
		Content:     req.Message,
		Priority:    2,
		ScheduledAt: &scheduledTime,
		Metadata:    convertMapToMetadata(req.Data),
	}

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

	// Set default pagination and prevents overload
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

	// we can get notification from service
	notifications, err := h.service.GetUserNotifications(ctx, req.UserId, int(limit), offset)
	if err != nil {
		log.Printf("Failed to get user notifications: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get notifications: %v", err))
	}

	// converting data into proto format
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

// marks the notification as read
func (h *GRPCHandler) MarkAsRead(ctx context.Context, req *pb.MarkAsReadRequest) (*pb.MarkAsReadResponse, error) {
	log.Printf("gRPC MarkAsRead called for notification: %s", req.NotificationId)

	if req.NotificationId == "" {
		return nil, status.Error(codes.InvalidArgument, "notification_id is required")
	}

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

// TO send push notification in a particular device register the device first
func (h *GRPCHandler) RegisterDevice(ctx context.Context, req *pb.RegisterDeviceRequest) (*pb.RegisterDeviceResponse, error) {
	log.Printf("gRPC RegisterDevice called for user: %s", req.UserId)

	if req.UserId == "" || req.DeviceToken == "" || req.Platform == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, device_token, and platform are required") // checks  for invalid entries again
	}

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

// sends a friend request notification
func (h *GRPCHandler) SendFriendRequest(ctx context.Context, req *pb.SendFriendRequestRequest) (*pb.SendFriendRequestResponse, error) {
	log.Printf("gRPC SendFriendRequest called from %s to %s", req.FromUserId, req.ToUserId)

	if req.FromUserId == "" || req.ToUserId == "" || req.FromUsername == "" {
		return nil, status.Error(codes.InvalidArgument, "from_user_id, to_user_id, and from_username are required")
	}

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

// health status of the service is checked here
func (h *GRPCHandler) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Status:    "healthy",
		Service:   "gosocial-notifications-grpc",
		Timestamp: timestamppb.New(time.Now()),
	}, nil
}

// Helper functions to map the data
func convertMapToMetadata(data map[string]string) common.NotificationMetadata {
	metadata := make(common.NotificationMetadata)
	for k, v := range data {
		metadata[k] = v
	}
	return metadata
}

// converts dbnotifications into a map
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
	return fmt.Sprintf("notif_%d", time.Now().UnixNano())
}
