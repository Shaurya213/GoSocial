package service

import (
	"context"
	"errors"
	"gosocial/internal/chat/repository"
	"gosocial/internal/dbmysql"
	"time"
)

type ChatService interface {
	SendMessage(ctx context.Context, msg *dbmysql.Message) (*dbmysql.Message, error)
	GetMessageHistory(ctx context.Context, conversationID string) ([]*dbmysql.Message, error)
}

type chatService struct {
	repo repository.ChatRepository
}

func NewChatService(r repository.ChatRepository) ChatService {
	return &chatService{repo: r}
}

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
	msg.SentAt = time.Now().UTC()
	err := s.repo.Save(ctx, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *chatService) GetMessageHistory(ctx context.Context, conversationID string) ([]*dbmysql.Message, error) {
	if conversationID == "" {
		return nil, errors.New("conversation ID is required")
	}

	return s.repo.FetchHistory(ctx, conversationID)
}
