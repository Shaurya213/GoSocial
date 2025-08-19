package main

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	feedpb "gosocial/api/v1/feed"
	"gosocial/internal/dbmongo"
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
	go func() {
		lis, err := net.Listen("tcp", ":"+app.Config.Server.FeedServicePort)
		if err != nil {
			log.Fatalf("Failed to listen on port %s: %v", app.Config.Server.FeedServicePort, err)
		}
		log.Printf("Feed Service (gRPC) running on port %s", app.Config.Server.FeedServicePort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// --- MongoDB + GridFS Media Storage Initialization ---
	mongoClient, err := dbmongo.NewMongoConnection(app.Config)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB for Media: %v", err)
	}
	mediaStore := dbmongo.NewMediaStorage(mongoClient) // Use your GridFS wrapper

	// Start HTTP Media server on :8080
	go func() {
		http.HandleFunc("/media/", func(w http.ResponseWriter, r *http.Request) {
			// Extract fileID from URL path
			fileID := strings.TrimPrefix(r.URL.Path, "/media/")
			if fileID == "" {
				http.Error(w, "File ID required", http.StatusBadRequest)
				return
			}

			reader, mediaFile, err := mediaStore.DownloadFile(r.Context(), fileID)
			if err != nil {
				log.Printf("Failed to download file %s: %v", fileID, err)
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}

			// Set proper headers
			w.Header().Set("Content-Type", getContentType(mediaFile.Filename))
			w.Header().Set("Content-Disposition", `inline; filename="`+mediaFile.Filename+`"`)
			_, err = io.Copy(w, reader)
			if err != nil {
				log.Printf("Failed to stream file %s: %v", fileID, err)
				http.Error(w, "Failed to serve file", http.StatusInternalServerError)
				return
			}
			log.Printf("Served media file: %s (%s)", mediaFile.Filename, fileID)
		})

		log.Printf("Media HTTP Server running on port 8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("Failed to serve Media HTTP: %v", err)
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
