package notif

import (
	"encoding/json"
	"fmt"
	"gosocial/internal/common"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type NotificationHandler struct { // NotificationHandler handles HTTP requests for notifications
	service *NotificationService
}

func NewNotificationHandler(service *NotificationService) *NotificationHandler {
	return &NotificationHandler{
		service: service,
	}
}

type SendNotificationRequest struct {
	UserID        string                      `json:"user_id" validate:"required"`
	Type          string                      `json:"type" validate:"required"`
	Header        string                      `json:"header" validate:"required"`
	Content       string                      `json:"content" validate:"required"`
	ImageURL      *string                     `json:"image_url,omitempty"`
	Priority      int                         `json:"priority" validate:"min=1,max=5"`
	TriggerUserID *string                     `json:"trigger_user_id,omitempty"`
	Metadata      common.NotificationMetadata `json:"metadata,omitempty"`
}

type ScheduleNotificationRequest struct {
	SendNotificationRequest
	ScheduledAt time.Time `json:"scheduled_at" validate:"required"`
}

type RegisterDeviceRequest struct {
	UserID      string `json:"user_id" validate:"required"`
	DeviceToken string `json:"device_token" validate:"required"`
	Platform    string `json:"platform" validate:"required,oneof=android ios web"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func (h *NotificationHandler) SendNotification(w http.ResponseWriter, r *http.Request) {
	var req SendNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := h.validateSendRequest(req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Validation failed", err)
		return
	}

	event := common.NotificationEvent{
		Type:          common.NotificationType(req.Type),
		UserID:        req.UserID,
		TriggerUserID: req.TriggerUserID,
		Header:        req.Header,
		Content:       req.Content,
		ImageURL:      req.ImageURL,
		Priority:      req.Priority,
		Metadata:      req.Metadata,
	}

	if err := h.service.SendNotification(r.Context(), event); err != nil {
		h.sendError(w, http.StatusInternalServerError, "Failed to send notification", err)
		return
	}

	h.sendSuccess(w, "Notification sent successfully", nil)
}

func (h *NotificationHandler) ScheduleNotification(w http.ResponseWriter, r *http.Request) {
	var req ScheduleNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := h.validateScheduleRequest(req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Validation failed", err)
		return
	}

	event := common.NotificationEvent{
		Type:          common.NotificationType(req.Type),
		UserID:        req.UserID,
		TriggerUserID: req.TriggerUserID,
		Header:        req.Header,
		Content:       req.Content,
		ImageURL:      req.ImageURL,
		Priority:      req.Priority,
		Metadata:      req.Metadata,
		ScheduledAt:   &req.ScheduledAt,
	}

	if err := h.service.ScheduleNotification(r.Context(), event); err != nil {
		h.sendError(w, http.StatusInternalServerError, "Failed to schedule notification", err)
		return
	}

	h.sendSuccess(w, "Notification scheduled successfully", map[string]interface{}{
		"scheduled_at": req.ScheduledAt,
	})
}

func (h *NotificationHandler) GetUserNotifications(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	if userID == "" {
		h.sendError(w, http.StatusBadRequest, "User ID is required", nil)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0 // default
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	notifications, err := h.service.GetUserNotifications(r.Context(), userID, limit, offset)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Failed to get notifications", err)
		return
	}

	h.sendSuccess(w, "Notifications retrieved successfully", notifications)
}

func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	notificationID := vars["notificationID"]

	if notificationID == "" {
		h.sendError(w, http.StatusBadRequest, "Notification ID is required", nil)
		return
	}

	userID := r.Header.Get("X-User-ID") // Adjust based on your auth system
	if userID == "" {
		h.sendError(w, http.StatusBadRequest, "User ID is required", nil)
		return
	}

	if err := h.service.MarkAsRead(r.Context(), notificationID, userID); err != nil {
		h.sendError(w, http.StatusInternalServerError, "Failed to mark notification as read", err)
		return
	}

	h.sendSuccess(w, "Notification marked as read", nil)
}

func (h *NotificationHandler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	var req RegisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := h.validateDeviceRequest(req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Validation failed", err)
		return
	}

	if err := h.service.RegisterDeviceToken(r.Context(), req.UserID, req.DeviceToken, req.Platform); err != nil {
		h.sendError(w, http.StatusInternalServerError, "Failed to register device", err)
		return
	}

	h.sendSuccess(w, "Device registered successfully", nil)
}

func (h *NotificationHandler) SendFriendRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromUserID     string `json:"from_user_id" validate:"required"`
		ToUserID       string `json:"to_user_id" validate:"required"`
		FromUserHandle string `json:"from_user_handle" validate:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := h.service.SendFriendRequestNotification(
		r.Context(),
		req.FromUserID,
		req.ToUserID,
		req.FromUserHandle,
	); err != nil {
		h.sendError(w, http.StatusInternalServerError, "Failed to send friend request notification", err)
		return
	}

	h.sendSuccess(w, "Friend request notification sent", nil)
}

func (h *NotificationHandler) validateSendRequest(req SendNotificationRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if req.Type == "" {
		return fmt.Errorf("type is required")
	}
	if req.Header == "" {
		return fmt.Errorf("header is required")
	}
	if req.Content == "" {
		return fmt.Errorf("content is required")
	}
	if req.Priority < 1 || req.Priority > 5 {
		return fmt.Errorf("priority must be between 1 and 5")
	}
	return nil
}

func (h *NotificationHandler) validateScheduleRequest(req ScheduleNotificationRequest) error {
	if err := h.validateSendRequest(req.SendNotificationRequest); err != nil {
		return err
	}
	if req.ScheduledAt.Before(time.Now()) {
		return fmt.Errorf("scheduled_at must be in the future")
	}
	return nil
}

func (h *NotificationHandler) validateDeviceRequest(req RegisterDeviceRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if req.DeviceToken == "" {
		return fmt.Errorf("device_token is required")
	}
	if req.Platform == "" {
		return fmt.Errorf("platform is required")
	}
	validPlatforms := map[string]bool{"android": true, "ios": true, "web": true}
	if !validPlatforms[req.Platform] {
		return fmt.Errorf("platform must be one of: android, ios, web")
	}
	return nil
}

func (h *NotificationHandler) sendSuccess(w http.ResponseWriter, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func (h *NotificationHandler) sendError(w http.ResponseWriter, statusCode int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Message: message,
		Error:   errorMsg,
	})
}
