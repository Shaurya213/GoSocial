// package main

// import (
// 	"log"
// 	"os"

// 	"github.com/joho/godotenv"
// 	"gorm.io/driver/mysql"
// 	"gorm.io/gorm"

// 	"gosocial/internal/config"
// 	"gosocial/internal/dbmysql"
// )

// func main() {

// 	if err := godotenv.Load(); err != nil {
// 		log.Println("No .env file found, using system environment variables")
// 	}

// 	if os.Getenv("DB_HOST") == "" || os.Getenv("DB_USER") == "" {
// 		log.Println("Warning: Some database environment variables might be missing")
// 		log.Printf("DB_HOST: '%s', DB_USER: '%s'", os.Getenv("DB_HOST"), os.Getenv("DB_USER"))
// 	}

// 	log.Println("Connecting to database...")
// 	db, err := gorm.Open(mysql.Open(config.DSN()), &gorm.Config{})
// 	if err != nil {
// 		log.Fatalf("Failed to connect to database: %v", err)
// 	}

// 	dbmysql.SetDB(db)
// 	log.Println("Database connection established successfully!")

// 	log.Println("Running database migration...")
// 	if err := db.AutoMigrate(&dbmysql.Notification{}); err != nil {
// 		log.Fatalf("Migration failed: %v", err)
// 	}

// 	log.Println("Database migration completed successfully!")
// 	log.Println("Application initialized successfully!")

// 	// Optional: Test the database connection
// 	sqlDB, err := db.DB()
// 	if err == nil {
// 		if err := sqlDB.Ping(); err != nil {
// 			log.Printf("Warning: Database ping failed: %v", err)
// 		} else {
// 			log.Println("Database ping successful!")
// 		}
// 	}

// 	log.Println("Application is ready to serve requests...")
// }

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gosocial/internal/wire"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize application with dependency injection
	log.Println("Initializing application...")
	app, err := wire.InitializeApplication()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Setup HTTP router
	router := setupRouter(app)

	// Create HTTP server
	server := &http.Server{
		Addr:           fmt.Sprintf("%s:%s", app.Config.Server.Host, app.Config.Server.Port),
		Handler:        router,
		ReadTimeout:    time.Duration(app.Config.Server.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(app.Config.Server.WriteTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown notification service
	if app.Service != nil {
		app.Service.Shutdown()
	}

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server gracefully stopped")
}

// setupRouter configures HTTP routes
func setupRouter(app *wire.Application) *mux.Router {
	router := mux.NewRouter()

	// Add CORS middleware
	router.Use(corsMiddleware)
	router.Use(loggingMiddleware)

	// API v1 routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Health check
	api.HandleFunc("/health", healthCheckHandler).Methods("GET")

	// Notification routes
	notifications := api.PathPrefix("/notifications").Subrouter()
	notifications.HandleFunc("/send", app.Handler.SendNotification).Methods("POST")
	notifications.HandleFunc("/schedule", app.Handler.ScheduleNotification).Methods("POST")
	notifications.HandleFunc("/users/{userID}", app.Handler.GetUserNotifications).Methods("GET")
	notifications.HandleFunc("/{notificationID}/read", app.Handler.MarkAsRead).Methods("PUT")
	notifications.HandleFunc("/device/register", app.Handler.RegisterDevice).Methods("POST")
	notifications.HandleFunc("/friend-request", app.Handler.SendFriendRequest).Methods("POST")

	return router
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// healthCheckHandler provides basic health check
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"gosocial-notifications"}`))
}
