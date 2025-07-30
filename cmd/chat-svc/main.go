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
	"gosocial/internal/config"
	"gosocial/internal/dbmysql"
	"gosocial/internal/di"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	err := godotenv.Load()

	log.Println("Starting Chat Service...")

	// Load configuration to set environment variables
	cfg := config.LoadConfig()
	
	// Set MYSQL_DSN environment variable that NewMySQL() expects
	if err := os.Setenv("MYSQL_DSN", cfg.GetMySQLDSN()); err != nil {
		log.Fatalf("Failed to set MYSQL_DSN: %v", err)
	}

	// Initialize all dependencies via Wire
	chatHandler, cleanup, err := di.InitializeChatService()
	if err != nil {
		log.Fatalf("Failed to initialize chat service: %v", err)
	}
	defer cleanup()
	// Auto-migrate the Message model (since your NewMySQL doesn't do this)
	db, err := dbmysql.NewMySQL(cfg)
	if err != nil {
		log.Fatalf("Failed to get database connection for migration: %v", err)
	}
	if err := db.AutoMigrate(&dbmysql.Message{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(loggingUnaryInterceptor),
		grpc.StreamInterceptor(loggingStreamInterceptor),
	)

	// Register services
	pb.RegisterChatServiceServer(grpcServer, chatHandler)
	reflection.Register(grpcServer)

	// Start server
	lis, err := net.Listen("tcp", ":"+cfg.Server.ChatServicePort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.Server.ChatServicePort, err)
	}

	// Graceful shutdown handling
	go func() {
		log.Printf("Chat Service running on port %s", cfg.Server.ChatServicePort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Chat Service...")
	grpcServer.GracefulStop()
	log.Println("Chat Service stopped")
}

func loggingUnaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
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

func loggingStreamInterceptor(
	srv interface{},
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	log.Printf("⟷ %s stream started", info.FullMethod)
	
	err := handler(srv, stream)
	
	if err != nil {
		log.Printf("✗ %s stream ended with error: %v", info.FullMethod, err)
	} else {
		log.Printf("✓ %s stream completed", info.FullMethod)
	}
	
	return err
}

