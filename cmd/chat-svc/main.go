package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "gosocial/api/v1/chat"
	"gosocial/internal/di"
	"gosocial/internal/dbmysql"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)
func main() {
	log.Println("Starting Chat Service...")

	// Get both handler and database
	app, cleanup, err := di.InitializeChatService()
	if err != nil {
		log.Fatalf("Failed to initialize chat service: %v", err)
	}
	defer cleanup()

	// Run migrations in main.go where they fuckingn belong
	if err := app.DB.AutoMigrate(&dbmysql.Message{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Println("✅ Database migration completed")

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(loggingUnaryInterceptor),
		grpc.StreamInterceptor(loggingStreamInterceptor),
	)
	// Register services using app.Handler
	pb.RegisterChatServiceServer(grpcServer, app.Handler)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":"+app.Config.Server.ChatServicePort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", app.Config.Server.ChatServicePort, err)
	}

	// Graceful shutdown handling
	go func() {
		log.Printf("Chat Service running on port %s", app.Config.Server.ChatServicePort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Chat Service...")
	grpcServer.GracefulStop()
	log.Println("Chat Service stopped")
}

func loggingUnaryInterceptor(ctx context.Context, req interface{}, 
info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	start := time.Now()
	log.Printf("→ %s", info.FullMethod)

	resp, err := handler(ctx, req)

	duration := time.Since(start)
	if err != nil {
		log.Printf("✗ %s failed (%v): %v", info.FullMethod, duration, err)
	} else {
		log.Printf("✓ %s completed (%v)", info.FullMethod, duration)
	}

	return resp, err
}

func loggingStreamInterceptor( srv interface{}, stream grpc.ServerStream,
info *grpc.StreamServerInfo, handler grpc.StreamHandler,) error { 

	log.Printf("⟷ %s stream started", info.FullMethod)
	err := handler(srv, stream)

	if err != nil {
		log.Printf("✗ %s stream ended with error: %v", info.FullMethod, err)
	} else {
		log.Printf("✓ %s stream completed", info.FullMethod)
	}
	return err
}

