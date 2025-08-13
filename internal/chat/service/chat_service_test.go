package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.uber.org/mock/gomock"


	"gosocial/internal/chat/service/mocks" 
	"gosocial/internal/dbmysql"
)

func TestChatService_SendMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// ✅ CORRECT: Use mocks.NewMockChatRepository
	mockRepo := mocks.NewMockChatRepository(ctrl) 
	service := NewChatService(mockRepo)

	tests := []struct {
		name        string
		message     *dbmysql.Message
		mockSetup   func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful message send",
			message: &dbmysql.Message{
				ConversationID: "conv-123",
				SenderID:       "user-456",
				Content:        "Hello, world!",
			},
			mockSetup: func() {
				mockRepo.EXPECT().
					Save(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, msg *dbmysql.Message) error {
						assert.WithinDuration(t, time.Now(), msg.SentAt, time.Second)
						return nil
					}).
					Times(1)
			},
			expectError: false,
		},
		{
			name: "empty conversation ID",
			message: &dbmysql.Message{
				ConversationID: "",
				SenderID:       "user-456",
				Content:        "Hello, world!",
			},
			mockSetup:   func() {},
			expectError: true,
			errorMsg:    "conversation ID cannot be empty",
		},
		{
			name: "repository save error",
			message: &dbmysql.Message{
				ConversationID: "conv-123",
				SenderID:       "user-456",
				Content:        "Hello, world!",
			},
			mockSetup: func() {
				mockRepo.EXPECT().
					Save(gomock.Any(), gomock.Any()).
					Return(errors.New("database connection failed")).
					Times(1)
			},
			expectError: true,
			errorMsg:    "database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			savedMsg, err := service.SendMessage(context.Background(), tt.message)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, savedMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, savedMsg)
				assert.WithinDuration(t, time.Now(), savedMsg.SentAt, time.Second)
			}
		})
	}
}

func TestChatService_GetMessageHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockChatRepository(ctrl)
	service := NewChatService(mockRepo)

	tests := []struct {
		name           string
		conversationID string
		mockSetup      func()
		expectedCount  int
		expectError    bool
		errorMsg       string
	}{
		{
			name:           "successful history fetch",
			conversationID: "conv-123",
			mockSetup: func() {
				messages := []*dbmysql.Message{
					{
						MessageID:      1,
						ConversationID: "conv-123",
						SenderID:       "user-456",
						Content:        "Hello",
						SentAt:         time.Now().Add(-10 * time.Minute),
					},
					{
						MessageID:      2,
						ConversationID: "conv-123",
						SenderID:       "user-789",
						Content:        "Hi there!",
						SentAt:         time.Now().Add(-5 * time.Minute),
					},
				}

				mockRepo.EXPECT().
					FetchHistory(gomock.Any(), "conv-123").
					Return(messages, nil).
					Times(1)
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:           "empty conversation ID",
			conversationID: "",
			mockSetup:      func() {},
			expectedCount:  0,
			expectError:    true,
			errorMsg:       "conversation ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			messages, err := service.GetMessageHistory(context.Background(), tt.conversationID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, messages)
			} else {
				assert.NoError(t, err)
				assert.Len(t, messages, tt.expectedCount)
			}
		})
	}
}

