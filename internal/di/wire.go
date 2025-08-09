//go:build wireinject
// +build wireinject
package di

import (
	"fmt"
	"gosocial/internal/chat/handler"
	"gosocial/internal/chat/repository"
	"gosocial/internal/chat/service"
	"gosocial/internal/config"
	"gosocial/internal/common"
	"gosocial/internal/dbmysql"
	"gosocial/internal/notif"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"context"
	"google.golang.org/api/option"
	
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/google/wire"
)

// ChatProviderSet contains all providers for chat service
var ChatProviderSet = wire.NewSet(
	config.LoadConfig,
	// Use existing database connection function
	dbmysql.NewMySQL,
	
	// Repository layer
	repository.NewChatRepository,
	
	// Service layer
	service.NewChatService,
	
	// Handler layer
	handler.NewChatHandler,
)

// InitializeChatService wires up all dependencies for the chat service
func InitializeChatService() (*handler.ChatHandler, func(), error) {
	wire.Build(ChatProviderSet)
	return nil, nil, nil
} 

type Application struct {
	Config  *config.Config
	DB      *gorm.DB
	Handler *notif.NotificationHandler
	Service *notif.NotificationService
}

func InitializeApplication() (*Application, error) {
	wire.Build(
		config.LoadConfig,
		ProvideDatabaseConnection,
		dbmysql.NewNotificationRepository,
		dbmysql.NewDeviceRepository,
		ProvideFirebaseApp,
		ProvideFirebaseMessaging,
		ProvideEmailService,
		notif.NewNotificationService,
		ProvideNotificationServiceInterface, // FIXED: Add interface provider
		notif.NewNotificationHandler,
		wire.Struct(new(Application), "*"),
	)
	return &Application{}, nil
}

// FIXED: Add this provider function to convert concrete service to interface
func ProvideNotificationServiceInterface(service *notif.NotificationService) notif.NotificationServiceInterface {
	return service
}

func ProvideDatabaseConnection(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DatabaseName,
	)

	log.Printf("Connecting to MySQL: %s:%s/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.DatabaseName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
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
	if !cfg.Firebase.Enabled || cfg.Firebase.CredentialsFilePath == "" {
		log.Println("Firebase disabled or credentials missing")
		return nil, nil
	}

	opt := option.WithCredentialsFile(cfg.Firebase.CredentialsFilePath)
	firebaseConfig := &firebase.Config{ProjectID: cfg.Firebase.ProjectID}

	app, err := firebase.NewApp(context.Background(), firebaseConfig, opt)
	if err != nil {
		log.Printf("Firebase init error: %v", err)
		return nil, nil
	}

	return app, nil
}

func ProvideFirebaseMessaging(app *firebase.App) (*messaging.Client, error) {
	if app == nil {
		log.Println("No Firebase app provided")
		return nil, nil
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		log.Printf("Failed to get FCM client: %v", err)
		return nil, nil
	}
	return client, nil
}

func ProvideEmailService(cfg *config.Config) common.EmailService {
	return &MockEmailService{}
}

type MockEmailService struct{}

func (m *MockEmailService) SendEmail(to, subject, body string) error {
	log.Printf("Mock Email - To: %s | Subject: %s", to, subject)
	return nil
}
