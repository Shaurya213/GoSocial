package main

import (
	"gosocial/api/v1/chat"
	//"gosocial/internal/common"
	"github.com/joho/godotenv"
	"google.golang.org/grpc/reflection"
	"gosocial/internal/dbmysql"
	"gosocial/internal/di"
	"log"
	"net"

	"google.golang.org/grpc"
)

func main() {
	err := godotenv.Load() // Your DB config
	if err != nil {
		log.Fatalf("❌ .env file not found: %v", err)
	}

	// Wire-injected handler
	db, err := dbmysql.NewMySQL()
	if err != nil {
		log.Fatalf("❌ Failed to connect DB: %v", err)
	}
	chatHandler := di.InitChatHandler(db)

	err = db.AutoMigrate(&dbmysql.Message{})
	if err != nil {
		log.Fatalf("❌ Failed to migrate DB: %v", err)
	}

	// Set up gRPC
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("❌ Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// Register chat service
	chat.RegisterChatServiceServer(grpcServer, chatHandler)

	reflection.Register(grpcServer)

	log.Println("✅ gRPC Chat Server running on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("❌ gRPC failed: %v", err)
	}
}
