package main

import (
	"context"
	"log"
	"net/http"

	"GoSocial/internal/config"
	"GoSocial/internal/dbmongo"
	"GoSocial/internal/media"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Connect to MongoDB using your existing connection
	mongoClient, err := dbmongo.NewMongoConnection(cfg)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer mongoClient.Close(context.Background())

	// Create HTTP server using your existing MediaStorage
	mediaServer := media.NewHTTPServer(mongoClient)

	// Start server
	log.Printf("🚀 Media HTTP Server starting on port 8080")
	log.Printf("📂 Serving files at: http://localhost:8080/media/{fileId}")

	if err := http.ListenAndServe(":8080", mediaServer); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
