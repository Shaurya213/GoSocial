package handler

import (
	"GoSocial/internal/chat/service"
	"GoSocial/internal/dbmysql"
	"context"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"

	//"gosocial/internal/chat/repository"
	//"google.golang.org/protobuf/types/known/timestamppb"

	pb "GoSocial/api/v1/chat" // Generated from your `chat.proto`
)

type ChatHandler struct {
	pb.UnimplementedChatServiceServer
	service service.ChatService
}

func NewChatHandler(svc service.ChatService) *ChatHandler {
	return &ChatHandler{service: svc}
}

// 1. SendMessage (Unary)

func (h *ChatHandler) SendMessages(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	// Convert proto → db model
	msg := &dbmysql.Message{
		ConversationID: req.ConversationId,
		SenderID:       req.SenderId,
		Content:        req.Content,
	}

	savedMsg, err := h.service.SendMessage(ctx, msg)
	if err != nil {
		return nil, err
	}

	// Convert db → proto
	resp := &pb.SendMessageResponse{
		Success: true,
		Message: &pb.ChatMessage{
			ConversationId: savedMsg.ConversationID,
			SenderId:       savedMsg.SenderID,
			Content:        savedMsg.Content,
			SentAt:         ToProtoTimestamp(savedMsg.SentAt),
		},
	}

	return resp, nil
}

// 2. GetChatHistory (Unary)

func (h *ChatHandler) GetChatHistory(ctx context.Context, req *pb.GetChatHistoryRequest) (*pb.GetChatHistoryResponse, error) {
	msgs, err := h.service.GetMessageHistory(ctx, req.ConversationId)
	if err != nil {
		return nil, err
	}

	var protoMsgs []*pb.ChatMessage
	for _, m := range msgs {
		protoMsgs = append(protoMsgs, &pb.ChatMessage{
			ConversationId: m.ConversationID,
			SenderId:       m.SenderID,
			Content:        m.Content,
			SentAt:         ToProtoTimestamp(m.SentAt),
		})
	}

	return &pb.GetChatHistoryResponse{
		Messages: protoMsgs,
	}, nil
}

func ToProtoTimestamp(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}
