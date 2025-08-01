// Package handler: this the chat handler
package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"

	pb "gosocial/api/v1/chat" 

	"gosocial/internal/chat/service"
	"gosocial/internal/dbmysql"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ChatHandler struct {
	pb.UnimplementedChatServiceServer
	chatService service.ChatService
	mu          sync.RWMutex
	streams     map[string][]pb.ChatService_StreamMessagesServer
}

func NewChatHandler(chatService service.ChatService) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
		streams: make(map[string][]pb.ChatService_StreamMessagesServer),
	}
}

//SendMessages is a method that exists
func (h *ChatHandler) SendMessages(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	domainMsg := &dbmysql.Message{
		ConversationID: req.ConversationId,
		SenderID: req.SenderId,
		Content: req.Content,
	}
	savedMsg, err := h.chatService.SendMessage(ctx, domainMsg)
	if err != nil {
		return nil, fmt.Errorf( "failed to send Message %v : internal codes : %v", err, codes.Internal )
	}

	protoMsg := &pb.ChatMessage{
		ConversationId: savedMsg.ConversationID,
		SenderId: savedMsg.SenderID,
		Content: savedMsg.Content,
		SentAt: timestamppb.New(savedMsg.SentAt),
	}

	h.broadcastToStream(savedMsg.ConversationID, protoMsg)

	return &pb.SendMessageResponse{
		Success: true,
		Message: protoMsg,
	}, nil
}

func (h *ChatHandler) GetChatHistory(ctx context.Context, req *pb.GetChatHistoryRequest) (*pb.GetChatHistoryResponse, error) {
	domainMessages, err := h.chatService.GetMessageHistory(ctx, req.ConversationId)
	if err != nil {
		return nil, fmt.Errorf("Failed to get chat history : %v, Error codes: %v", err, codes.Internal)
	}

	protoMessages := make([]*pb.ChatMessage, 0, len(domainMessages))
	start := int(req.Offset)
	end := start + int(req.Limit)

	if start >= len(domainMessages) {
		return &pb.GetChatHistoryResponse{Messages: []*pb.ChatMessage{}}, nil
	}

	if end > len(domainMessages) {
		end = len(domainMessages) 
	}

	for _, msg := range domainMessages[start:end] {
		protoMessage := &pb.ChatMessage{
			ConversationId: msg.ConversationID,
			SenderId: msg.SenderID,
			Content: msg.Content,
			SentAt: timestamppb.New(msg.SentAt),
		}
		protoMessages = append(protoMessages, protoMessage)
	}

	return &pb.GetChatHistoryResponse{
		Messages: protoMessages,
	}, nil
}

func (h *ChatHandler) StreamMessages(stream pb.ChatService_StreamMessagesServer) error {
	var conversationID string

	go func(){
		for {
			protoMsg, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("Error reciving Messages: %v", err)
				break
			}
			if conversationID == "" {
				conversationID = protoMsg.ConversationId
				h.mu.Lock()
				h.streams[conversationID] = append(h.streams[conversationID], stream)
				h.mu.Unlock()
			}

			domainMsg := &dbmysql.Message{
				ConversationID: protoMsg.ConversationId,
				SenderID: protoMsg.SenderId,
				Content: protoMsg.Content,
			} 

			savedMsg, err := h.chatService.SendMessage(stream.Context(), domainMsg)

			if err != nil {
				log.Printf("Failed to save Steamed Messages: %v", err)
				continue
			}

			brodcastMsg := &pb.ChatMessage{
				ConversationId: savedMsg.ConversationID,
				SenderId: savedMsg.SenderID,
				Content: savedMsg.Content,
				SentAt: timestamppb.New(savedMsg.SentAt),
			}
			h.broadcastToStream(savedMsg.ConversationID, brodcastMsg)
		}
	}()

	select {
	case <-stream.Context().Done():
		h.removeStream(conversationID, stream)
		return stream.Context().Err()
	}
}

func (h *ChatHandler) broadcastToStream(conversationID string, msg *pb.ChatMessage) {
	h.mu.RLock()
	streams, ok := h.streams[conversationID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	for _, stream := range streams {
		if err := stream.Send(msg); err != nil {
			log.Printf("Failed to print stream: %v", err)
			//TODO: Remove failed steams
		}
	}
}

func (h *ChatHandler) removeStream(conversationID string, stream pb.ChatService_StreamMessagesServer) {
	h.mu.Lock()
	defer h.mu.Unlock()

	streams, ok := h.streams[conversationID]
	if !ok {
		return
	}
	for i, st := range streams {
		if st == stream {
			h.streams[conversationID] = append(streams[:i], streams[i+1:]...)
			break
		}
	}
}

