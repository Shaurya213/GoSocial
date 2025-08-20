package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	feedpb "gosocial/api/v1/feed"
	"gosocial/internal/dbmysql"
	"gosocial/internal/di"
	//"gosocial/internal/dbmongo/media_storage.go" // <--- correct import

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// If your dependency injection does NOT expose MediaStorage, you can create it directly here:

func main() {
	log.Println("Starting Feed Service...")

	// Initialize feed service using wire
	app, cleanup, err := di.InitializeFeedService()
	if err != nil {
		log.Fatalf("Failed to initialize feed service: %v", err)
	}
	defer cleanup()

	// Run migrations:
	if err := app.DB.AutoMigrate(
		&dbmysql.Content{},
		&dbmysql.MediaRef{},
		&dbmysql.Reaction{},
		&dbmysql.User{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("✅ Database migration completed")

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(loggingUnaryInterceptor),
	)
	feedpb.RegisterFeedServiceServer(grpcServer, app.Handler)
	reflection.Register(grpcServer)

	// Start gRPC server

	lis, err := net.Listen("tcp", ":"+app.Config.Server.FeedServicePort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", app.Config.Server.FeedServicePort, err)
	}

	go func() {
		log.Printf("Feed Service running on port %s", app.Config.Server.FeedServicePort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Feed Service...")
	grpcServer.GracefulStop()
	log.Println("Feed Service stopped")
}

// Simple content-type helper, can extend for more types
func getContentType(filename string) string {
	if strings.HasSuffix(strings.ToLower(filename), ".png") {
		return "image/png"
	}
	if strings.HasSuffix(strings.ToLower(filename), ".jpg") || strings.HasSuffix(strings.ToLower(filename), ".jpeg") {
		return "image/jpeg"
	}
	if strings.HasSuffix(strings.ToLower(filename), ".mp4") {
		return "video/mp4"
	}
	return "application/octet-stream"
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
