package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "gosocial/api/v1/chat"
	"gosocial/internal/chat/handler"
	"gosocial/internal/chat/repository"
	"gosocial/internal/chat/service"
	"gosocial/internal/dbmysql"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	defaultPort = "7003" 
	defaultDBHost = "localhost"
	defaultDBPort = "3306"
	defaultDBUser = "root"
	defaultDBPassword = "password"
	defaultDBName = "gosocial"
)

func main() {
	log.Println("Starting Chat Service...")

	// Get configuration from environment variables or use defaults
	port := getEnv("CHAT_SERVICE_PORT", defaultPort)
	dbHost := getEnv("DB_HOST", defaultDBHost)
	dbPort := getEnv("DB_PORT", defaultDBPort)
	dbUser := getEnv("DB_USER", defaultDBUser)
	dbPassword := getEnv("DB_PASSWORD", defaultDBPassword)
	dbName := getEnv("DB_NAME", defaultDBName)

	// Initialize database connection
	db, err := initDatabase(dbHost, dbPort, dbUser, dbPassword, dbName)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate the Message model
	if err := db.AutoMigrate(&dbmysql.Message{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Initialize dependency injection layers
	chatRepo := repository.NewChatRepository(db)
	chatService := service.NewChatService(chatRepo)
	chatHandler := handler.NewChatHandler(chatService)

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(loggingUnaryInterceptor),
		grpc.StreamInterceptor(loggingStreamInterceptor),
	)

	// Register chat service
	pb.RegisterChatServiceServer(grpcServer, chatHandler)

	// Enable gRPC reflection for testing tools like grpcurl
	reflection.Register(grpcServer)

	// Start server in a goroutine
	go func() {
		lis, err := net.Listen("tcp", ":"+port)
		if err != nil {
			log.Fatalf("Failed to listen on port %s: %v", port, err)
		}

		log.Printf("Chat Service is running on port %s", port)
		log.Printf("gRPC reflection enabled for testing")
		
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Chat Service...")

	// Graceful shutdown
	grpcServer.GracefulStop()
	
	// Close database connection
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Close()
	}

	log.Println("Chat Service stopped")
}

// initDatabase initializes the MySQL database connection
func initDatabase(host, port, user, password, dbname string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, password, host, port, dbname)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	// Configure connection pool
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	return db, nil
}

// getEnv gets environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// loggingUnaryInterceptor logs unary RPC calls
func loggingUnaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	log.Printf("Received unary call: %s", info.FullMethod)
	
	resp, err := handler(ctx, req)
	
	duration := time.Since(start)
	if err != nil {
		log.Printf("Unary call %s failed after %v: %v", info.FullMethod, duration, err)
	} else {
		log.Printf("Unary call %s completed successfully in %v", info.FullMethod, duration)
	}
	
	return resp, err
}

// loggingStreamInterceptor logs streaming RPC calls
func loggingStreamInterceptor(
	srv interface{},
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	log.Printf("Received stream call: %s", info.FullMethod)
	
	err := handler(srv, stream)
	
	if err != nil {
		log.Printf("Stream call %s ended with error: %v", info.FullMethod, err)
	} else {
		log.Printf("Stream call %s completed successfully", info.FullMethod)
	}
	
	return err
}

