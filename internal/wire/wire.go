//go:build wireinject
// +build wireinject

package wire

import (
	"context"
	"fmt"
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

type Application struct {
	Config  *config.Config
	DB      *gorm.DB
	Handler *notif.NotificationHandler
	Service *notif.NotificationService
}

func InitializeApplication() (*Application, error) {
	wire.Build(
		ProvideConfig,
		ProvideDatabaseConnection, //  receives config as parameter
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

func ProvideConfig() *config.Config {
	return &config.Config{
		Database: config.DatabaseConfig{
			Host:         getEnvOrDefault("DB_HOST", "192.168.63.59"),
			Port:         getEnvOrDefault("DB_PORT", "3306"),
			Username:     getEnvOrDefault("DB_USER", "gosocial_user"),
			Password:     getEnvOrDefault("DB_PASSWORD", "G0Social@123"),
			DatabaseName: getEnvOrDefault("DB_NAME", "gosocial_db"),
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

func ProvideDatabaseConnection(cfg *config.Config) (*gorm.DB, error) {
	// Build DSN with proper validation
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DatabaseName,
	)

	log.Printf("Attempting to connect to database: %s:%s/%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.DatabaseName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	dbmysql.SetDB(db)

	if err := db.AutoMigrate(
		&dbmysql.Notification{},
		&dbmysql.Device{},
	); err != nil {
		log.Printf("Migration warning: %v", err)
	}

	return db, nil
}

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
		return nil, nil
	}

	return app, nil
}

func ProvideFirebaseMessaging(app *firebase.App) (*messaging.Client, error) {
	if app == nil {
		log.Println("Firebase app not available, FCM disabled")
		return nil, nil
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		log.Printf("Failed to create FCM client: %v", err)
		return nil, nil
	}

	return client, nil
}

func ProvideEmailService(cfg *config.Config) common.EmailService {
	return &MockEmailService{}
}

type MockEmailService struct{}

func (m *MockEmailService) SendEmail(to, subject, body string) error {
	log.Printf("Mock Email - To: %s, Subject: %s", to, subject)
	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
