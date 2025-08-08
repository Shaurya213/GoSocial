package handler

import (
    "context"
    "io"
    "testing"
    "time"

    "go.uber.org/mock/gomock"
    "github.com/stretchr/testify/assert"
    "google.golang.org/grpc"
    "google.golang.org/grpc/test/bufconn"
    "google.golang.org/protobuf/types/known/timestamppb"

    pb "gosocial/api/v1/chat"
    "gosocial/internal/chat/handler/mocks"
    "gosocial/internal/dbmysql"
)

const bufSize = 1024 * 1024

func setupGRPCTest(t *testing.T) (pb.ChatServiceClient, *mocks.MockChatService, func()) {
    // Create in-memory gRPC server
    lis := bufconn.Listen(bufSize)
    
    ctrl := gomock.NewController(t)
    mockService := mocks.NewMockChatService(ctrl)
    
    // Create handler with mock service
    handler := NewChatHandler(mockService)
    
    // Create gRPC server
    s := grpc.NewServer()
    pb.RegisterChatServiceServer(s, handler)
    
    // Start server in background
    go func() {
        if err := s.Serve(lis); err != nil {
            t.Errorf("Server exited with error: %v", err)
        }
    }()
    
    // Create client connection
    ctx := context.Background()
    conn, err := grpc.DialContext(ctx, "bufnet", 
        grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
            return lis.Dial()
        }), grpc.WithInsecure())
    if err != nil {
        t.Fatalf("Failed to dial bufnet: %v", err)
    }
    
    client := pb.NewChatServiceClient(conn)
    
    cleanup := func() {
        conn.Close()
        s.Stop()
        ctrl.Finish()
    }
    
    return client, mockService, cleanup
}

func TestChatHandler_StreamMessages_RealGRPC(t *testing.T) {
    client, mockService, cleanup := setupGRPCTest(t)
    defer cleanup()

    t.Run("successful_streaming_workflow", func(t *testing.T) {
        // Mock the service calls
        mockService.EXPECT().
            SendMessage(gomock.Any(), gomock.Any()).
            DoAndReturn(func(ctx context.Context, msg *dbmysql.Message) (*dbmysql.Message, error) {
                return &dbmysql.Message{
                    MessageID: 1,
                    ConversationID: msg.ConversationID,
                    SenderID: msg.SenderID, 
                    Content: msg.Content,
                    SentAt: time.Now(),
                }, nil
            }).
            AnyTimes()

        // Create streaming client
        stream, err := client.StreamMessages(context.Background())
        assert.NoError(t, err)

        // Send a message
        err = stream.Send(&pb.ChatMessage{
            ConversationId: "conv-123",
            SenderId: "user-456",
            Content: "Hello from stream!",
            SentAt: timestamppb.New(time.Now()),
        })
        assert.NoError(t, err)

        // Receive the broadcasted message
        received, err := stream.Recv()
        assert.NoError(t, err)
        assert.Equal(t, "conv-123", received.ConversationId)
        assert.Equal(t, "Hello from stream!", received.Content)

        // Close the stream
        stream.CloseSend()
    })

    t.Run("stream_context_cancellation", func(t *testing.T) {
        ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
        defer cancel()

        stream, err := client.StreamMessages(ctx)
        assert.NoError(t, err)

        // Wait for context cancellation
        time.Sleep(150 * time.Millisecond)

        // Try to receive - should get context error
        _, err = stream.Recv()
        assert.Error(t, err)
    })
}

