package service

import (
	"GoSocial/internal/chat/repository"
	"GoSocial/internal/dbmysql"
	"context"
	"errors"
	"time"
)

// ChatService defines the interface exposed to the handler layer
type ChatService interface {
	SendMessage(ctx context.Context, msg *dbmysql.Message) (*dbmysql.Message, error)
	GetMessageHistory(ctx context.Context, conversationID string) ([]*dbmysql.Message, error)
}

type chatService struct {
	repo repository.ChatRepository
}

// Constructor used in DI/wire
func NewChatService(r repository.ChatRepository) ChatService {
	return &chatService{repo: r}
}

// SendMessage handles message validation and saving
func (s *chatService) SendMessage(ctx context.Context, msg *dbmysql.Message) (*dbmysql.Message, error) {
	// Input Validation
	if msg.ConversationID == "" {
		return nil, errors.New("conversation ID cannot be empty")
	}
	if msg.SenderID == "" {
		return nil, errors.New("sender ID cannot be empty")
	}
	if msg.Content == "" {
		return nil, errors.New("message content cannot be empty")
	}

	// Set server-side timestamp
	msg.SentAt = time.Now().UTC()

	// Save to DB via repository
	err := s.repo.Save(ctx, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// GetMessageHistory returns full message history of a conversation
func (s *chatService) GetMessageHistory(ctx context.Context, conversationID string) ([]*dbmysql.Message, error) {
	if conversationID == "" {
		return nil, errors.New("conversation ID is required")
	}

	return s.repo.FetchHistory(ctx, conversationID)
}
