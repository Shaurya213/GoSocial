package chat

import (
	"context"
	_ "time"

	chatpb "GoSocial/api/v1/chat"
	"GoSocial/internal/chat/models"
	"GoSocial/internal/chat/repository"

	_ "google.golang.org/protobuf/types/known/timestamppb"
)

// ChatService defines the business logic interface
type ChatService interface {
	SendMessage(ctx context.Context, req *chatpb.SendMessageRequest) (*chatpb.SendMessageResponse, error)
	GetChatHistory(ctx context.Context, req *chatpb.GetChatHistoryRequest) (*chatpb.GetChatHistoryResponse, error)
	SaveStreamedMessage(ctx context.Context, msg *chatpb.ChatMessage) error
}

// concrete service struct
type chatService struct {
	repo repository.ChatRepository
}

// NewChatService creates a new instance of ChatService
func NewChatService(repo repository.ChatRepository) ChatService {
	return &chatService{repo: repo}
}
