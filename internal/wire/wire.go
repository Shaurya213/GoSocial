//go:build wireinject
// +build wireinject

package wire

import (
	"context"
	"gosocial/internal/common"
	"gosocial/internal/config"
	"gosocial/internal/dbmysql"
	"gosocial/internal/notif"
	"log"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/google/wire"
	"google.golang.org/api/option"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Application holds all dependencies
type Application struct {
	Config  *config.Config
	DB      *gorm.DB
	Handler *notif.NotificationHandler
	Service *notif.NotificationService
}

// InitializeApplication wires up all dependencies
func InitializeApplication() (*Application, error) {
	wire.Build(
		ProvideConfig,
		ProvideDatabaseConnection,
		dbmysql.NewNotificationRepository,
		dbmysql.NewDeviceRepository,
		ProvideFirebaseApp,
		ProvideFirebaseMessaging,
		ProvideEmailService,
		notif.NewNotificationService,
		notif.NewNotificationHandler,
		wire.Struct(new(Application), "*"),
	)
	return &Application{}, nil
}

// ProvideConfig creates application configuration
func ProvideConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port:         getEnvOrDefault("SERVER_PORT", "8080"),
			Host:         getEnvOrDefault("SERVER_HOST", "0.0.0.0"),
			ReadTimeout:  30,
			WriteTimeout: 30,
			Environment:  getEnvOrDefault("ENVIRONMENT", "development"),
		},
		Database: config.DatabaseConfig{
			Host:         getEnvOrDefault("DB_HOST", "localhost"),
			Port:         getEnvOrDefault("DB_PORT", "3306"),
			Username:     getEnvOrDefault("DB_USER", "root"),
			Password:     getEnvOrDefault("DB_PASSWORD", ""),
			DatabaseName: getEnvOrDefault("DB_NAME", "gosocial"),
			MaxOpenConns: 25,
			MaxIdleConns: 5,
		},
		Firebase: config.FirebaseConfig{
			ProjectID:           getEnvOrDefault("FIREBASE_PROJECT_ID", ""),
			CredentialsFilePath: getEnvOrDefault("FIREBASE_CREDENTIALS_PATH", ""),
			Enabled:             getEnvOrDefault("FIREBASE_ENABLED", "false") == "true",
		},
		Notification: config.NotificationConfig{
			Workers:                5,
			ChannelBufferSize:      1000,
			ScheduledCheckInterval: 1,
			MaxRetries:             3,
			RetryDelay:             5,
			Enabled:                true,
		},
		Email: config.EmailConfig{
			SMTPHost:  getEnvOrDefault("SMTP_HOST", ""),
			SMTPPort:  587,
			Username:  getEnvOrDefault("SMTP_USERNAME", ""),
			Password:  getEnvOrDefault("SMTP_PASSWORD", ""),
			FromEmail: getEnvOrDefault("FROM_EMAIL", ""),
			FromName:  getEnvOrDefault("FROM_NAME", "GoSocial"),
			Enabled:   getEnvOrDefault("EMAIL_ENABLED", "false") == "true",
			UseTLS:    true,
		},
		Logging: config.LoggingConfig{
			Level:      getEnvOrDefault("LOG_LEVEL", "info"),
			Format:     getEnvOrDefault("LOG_FORMAT", "text"),
			OutputPath: getEnvOrDefault("LOG_OUTPUT", "stdout"),
		},
	}
}

// ProvideDatabaseConnection creates database connection
func ProvideDatabaseConnection() (*gorm.DB, error) {
	dsn := config.DSN()
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Set global db instance
	dbmysql.SetDB(db)

	// Auto migrate
	if err := db.AutoMigrate(
		&dbmysql.Notification{},
		&dbmysql.Device{},
	); err != nil {
		log.Printf("Migration warning: %v", err)
	}

	return db, nil
}

// ProvideFirebaseApp creates Firebase app instance (can return nil)
func ProvideFirebaseApp(cfg *config.Config) (*firebase.App, error) {
	if !cfg.Firebase.Enabled {
		log.Println("Firebase disabled")
		return nil, nil
	}

	if cfg.Firebase.CredentialsFilePath == "" {
		log.Println("Firebase credentials not provided")
		return nil, nil
	}

	opt := option.WithCredentialsFile(cfg.Firebase.CredentialsFilePath)
	firebaseConfig := &firebase.Config{
		ProjectID: cfg.Firebase.ProjectID,
	}

	app, err := firebase.NewApp(context.Background(), firebaseConfig, opt)
	if err != nil {
		log.Printf("Firebase initialization failed: %v", err)
		return nil, nil // Return nil instead of error to make it optional
	}

	return app, nil
}

// ProvideFirebaseMessaging creates Firebase messaging client (can return nil)
func ProvideFirebaseMessaging(app *firebase.App) (*messaging.Client, error) {
	if app == nil {
		log.Println("Firebase app not available, FCM disabled")
		return nil, nil // Return nil, nil to make it optional
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		log.Printf("Failed to create FCM client: %v", err)
		return nil, nil // Return nil, nil to make it optional
	}

	return client, nil
}

// ProvideEmailService creates email service
func ProvideEmailService(cfg *config.Config) common.EmailService {
	return &MockEmailService{}
}

// MockEmailService implements common.EmailService interface
type MockEmailService struct{}

func (m *MockEmailService) SendEmail(to, subject, body string) error {
	log.Printf("Mock Email - To: %s, Subject: %s", to, subject)
	return nil
}

// Helper function to get environment variables with defaults
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
