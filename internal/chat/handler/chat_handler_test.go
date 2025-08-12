package handler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "gosocial/api/v1/chat"
	"gosocial/internal/chat/handler/mocks"
	"gosocial/internal/dbmysql"
)

func TestChatHandler_SendMessages_Complete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockChatService(ctrl)
	handler := NewChatHandler(mockService)

	tests := []struct {
		name        string
		request     *pb.SendMessageRequest
		mockSetup   func()
		expectError bool
		checkResult func(*pb.SendMessageResponse)
	}{
		{
			name: "successful_message_send",
			request: &pb.SendMessageRequest{
				ConversationId: "conv-123",
				SenderId:       "user-456",
				Content:        "Hello World!",
			},
			mockSetup: func() {
				returnedMsg := &dbmysql.Message{
					MessageID:      1,
					ConversationID: "conv-123",
					SenderID:       "user-456",
					Content:        "Hello World!",
					SentAt:         time.Now().UTC(),
					Status:         "delivered",
				}

				mockService.EXPECT().
					SendMessage(gomock.Any(), gomock.Any()).
					Return(returnedMsg, nil).
					Times(1)
			},
			expectError: false,
			checkResult: func(resp *pb.SendMessageResponse) {
				assert.True(t, resp.Success)
				assert.NotNil(t, resp.Message)
				assert.Equal(t, "conv-123", resp.Message.ConversationId)
			},
		},
		{
			name: "service_error_handling",
			request: &pb.SendMessageRequest{
				ConversationId: "conv-123",
				SenderId:       "user-456",
				Content:        "Hello",
			},
			mockSetup: func() {
				mockService.EXPECT().
					SendMessage(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name: "empty_fields_handling",
			request: &pb.SendMessageRequest{
				ConversationId: "",
				SenderId:       "user-456",
				Content:        "Hello",
			},
			mockSetup: func() {
				mockService.EXPECT().
					SendMessage(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("conversation ID cannot be empty")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := handler.SendMessages(context.Background(), tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, resp)
				tt.checkResult(resp)
			}
		})
	}
}

func TestChatHandler_GetChatHistory_Complete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockChatService(ctrl)
	handler := NewChatHandler(mockService)

	sampleMessages := []*dbmysql.Message{
		{MessageID: 1, ConversationID: "conv-123", SenderID: "user-1", Content: "Msg1", SentAt: time.Now()},
		{MessageID: 2, ConversationID: "conv-123", SenderID: "user-2", Content: "Msg2", SentAt: time.Now()},
		{MessageID: 3, ConversationID: "conv-123", SenderID: "user-1", Content: "Msg3", SentAt: time.Now()},
	}

	tests := []struct {
		name        string
		request     *pb.GetChatHistoryRequest
		mockSetup   func()
		expectError bool
		checkResult func(*pb.GetChatHistoryResponse)
	}{
		{
			name: "successful_pagination",
			request: &pb.GetChatHistoryRequest{
				ConversationId: "conv-123",
				Limit:          2,
				Offset:         0,
			},
			mockSetup: func() {
				mockService.EXPECT().
					GetMessageHistory(gomock.Any(), "conv-123").
					Return(sampleMessages, nil).
					Times(1)
			},
			expectError: false,
			checkResult: func(resp *pb.GetChatHistoryResponse) {
				assert.Len(t, resp.Messages, 2)
			},
		},
		{
			name: "offset_beyond_messages",
			request: &pb.GetChatHistoryRequest{
				ConversationId: "conv-123",
				Limit:          10,
				Offset:         50,
			},
			mockSetup: func() {
				mockService.EXPECT().
					GetMessageHistory(gomock.Any(), "conv-123").
					Return(sampleMessages, nil).
					Times(1)
			},
			expectError: false,
			checkResult: func(resp *pb.GetChatHistoryResponse) {
				assert.Empty(t, resp.Messages)
			},
		},
		{
			name: "negative_values",
			request: &pb.GetChatHistoryRequest{
				ConversationId: "conv-123",
				Limit:          -5,
				Offset:         -10,
			},
			mockSetup: func() {
				mockService.EXPECT().
					GetMessageHistory(gomock.Any(), "conv-123").
					Return(sampleMessages, nil).
					Times(1)
			},
			expectError: false,
			checkResult: func(resp *pb.GetChatHistoryResponse) {
				assert.Empty(t, resp.Messages)
			},
		},
		{
			name: "service_error",
			request: &pb.GetChatHistoryRequest{
				ConversationId: "conv-123",
				Limit:          10,
				Offset:         0,
			},
			mockSetup: func() {
				mockService.EXPECT().
					GetMessageHistory(gomock.Any(), "conv-123").
					Return(nil, errors.New("database error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name: "zero_limit",
			request: &pb.GetChatHistoryRequest{
				ConversationId: "conv-123",
				Limit:          0,
				Offset:         0,
			},
			mockSetup: func() {
				mockService.EXPECT().
					GetMessageHistory(gomock.Any(), "conv-123").
					Return(sampleMessages, nil).
					Times(1)
			},
			expectError: false,
			checkResult: func(resp *pb.GetChatHistoryResponse) {
				assert.Empty(t, resp.Messages)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := handler.GetChatHistory(context.Background(), tt.request)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, resp)
				if tt.checkResult != nil {
					tt.checkResult(resp)
				}
			}
		})
	}
}

func TestChatHandler_BroadcastToStream_BusinessLogic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockChatService(ctrl)
	handler := NewChatHandler(mockService)

	t.Run("broadcast_to_nonexistent_conversation", func(t *testing.T) {
		msg := &pb.ChatMessage{
			ConversationId: "nonexistent",
			SenderId:       "user-456",
			Content:        "Hello",
			SentAt:         timestamppb.New(time.Now()),
		}

		// Should not panic when conversation doesn't exist
		assert.NotPanics(t, func() {
			handler.broadcastToStream("nonexistent", msg)
		})
	})

	t.Run("broadcast_with_empty_conversation_id", func(t *testing.T) {
		msg := &pb.ChatMessage{
			ConversationId: "",
			SenderId:       "user-456",
			Content:        "Hello",
			SentAt:         timestamppb.New(time.Now()),
		}

		assert.NotPanics(t, func() {
			handler.broadcastToStream("", msg)
		})
	})
}

func TestChatHandler_RemoveStream_BusinessLogic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockChatService(ctrl)
	handler := NewChatHandler(mockService)

	t.Run("remove_from_nonexistent_conversation", func(t *testing.T) {
		assert.NotPanics(t, func() {
			handler.removeStream("nonexistent", nil)
		})
	})

	t.Run("remove_with_empty_id", func(t *testing.T) {
		assert.NotPanics(t, func() {
			handler.removeStream("", nil)
		})
	})
}

func TestChatHandler_ConcurrentAccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockChatService(ctrl)
	handler := NewChatHandler(mockService)

	t.Run("concurrent_operations", func(t *testing.T) {
		var wg sync.WaitGroup
		numOps := 50

		for i := 0; i < numOps; i++ {
			wg.Add(2)

			go func() {
				defer wg.Done()
				msg := &pb.ChatMessage{
					ConversationId: "test-conv",
					SenderId:       "user-1",
					Content:        "test",
					SentAt:         timestamppb.New(time.Now()),
				}
				handler.broadcastToStream("test-conv", msg)
			}()

			go func() {
				defer wg.Done()
				handler.removeStream("test-conv", nil)
			}()
		}

		wg.Wait()
		// Should complete without race conditions
	})
}

