//go:build wireinject
// +build wireinject

package di

import (
	"fmt"

	"github.com/google/wire"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"

	//feedpb "gosocial/api/v1/feed"
	userpb "gosocial/api/v1/user"
	"gosocial/internal/config"
	"gosocial/internal/dbmongo"
	"gosocial/internal/dbmysql"
	"gosocial/internal/feed"
)

//// CHATS
//type ChatApp struct {
//    Handler *handler.ChatHandler
//    DB      *gorm.DB
//    Config  *config.Config
//}
//
//var ChatProviderSet = wire.NewSet(
//    config.LoadConfig,
//    dbmysql.NewMySQL,
//    repository.NewChatRepository,
//    service.NewChatService,
//    handler.NewChatHandler,
//    wire.Struct(new(ChatApp), "*"), // Wire creates ChatApp with all fields
//)
//
//// InitializeChatService now returns ChatApp with both handler and DB
//func InitializeChatService() (*ChatApp, func(), error) {
//    wire.Build(ChatProviderSet)
//    return nil, nil, nil
//}

// FEED SERVICE
type FeedApp struct {
	Handler      *feed.FeedHandlers
	Service      *feed.FeedService
	Config       *config.Config
	DB           *gorm.DB
	MediaStorage *dbmongo.MediaStorage // Add this
}

// Provider Functions

// Provide FeedRepository
func ProvideFeedRepository(db *gorm.DB, mediaStorage *dbmongo.MediaStorage) *feed.FeedRepository {
	return feed.NewFeedRepository(db, mediaStorage)
}

// Provide User Service Client
func ProvideUserServiceClient(cfg *config.Config) (userpb.UserServiceClient, func(), error) {
	conn, err := grpc.Dial(
		fmt.Sprintf("localhost:%s", cfg.Server.UserServicePort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}

	client := userpb.NewUserServiceClient(conn)
	cleanup := func() {
		conn.Close()
	}

	return client, cleanup, nil
}

// Provide FeedService
func ProvideFeedService(
	repo *feed.FeedRepository,
	userClient userpb.UserServiceClient,
) *feed.FeedService {
	return feed.NewFeedService(repo, repo, repo, userClient)
}

// Provide FeedHandlers
func ProvideFeedHandlers(feedService *feed.FeedService) *feed.FeedHandlers {
	return &feed.FeedHandlers{
		FeedSvc: feedService,
	}
}

// Provider Set
var FeedProviderSet = wire.NewSet(
	config.LoadConfig,
	dbmysql.NewMySQL,
	dbmongo.NewMongoConnection,
	dbmongo.NewMediaStorage,
	ProvideFeedRepository,
	ProvideUserServiceClient,
	ProvideFeedService,
	ProvideFeedHandlers,
	wire.Struct(new(FeedApp), "*"),
)

// Wire Entry Point
func InitializeFeedService() (*FeedApp, func(), error) {
	wire.Build(FeedProviderSet)
	return nil, nil, nil
}

//type Application struct {
//	Config  *config.Config
//	DB      *gorm.DB
//	Handler *notif.NotificationHandler
//	Service *notif.NotificationService
//}
//
//func InitializeApplication() (*Application, error) {
//	wire.Build(
//		config.LoadConfig,
//		//ProvideDatabaseConnection,
//		dbmysql.NewMySQL,
//		dbmysql.NewNotificationRepository,
//		dbmysql.NewDeviceRepository,
//		ProvideFirebaseApp,
//		ProvideFirebaseMessaging,
//		ProvideEmailService,
//		notif.NewNotificationService,
//		ProvideNotificationServiceInterface, // FIXED: Add interface provider
//		notif.NewNotificationHandler,
//		wire.Struct(new(Application), "*"),
//	)
//	return &Application{}, nil
//}
//
//// FIXED: Add this provider function to convert concrete service to interface
//func ProvideNotificationServiceInterface(service *notif.NotificationService) notif.NotificationServiceInterface {
//	return service
//}
//
///*
//func ProvideDatabaseConnection(cfg *config.Config) (*gorm.DB, error) {
//	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
//		cfg.Database.Username,
//		cfg.Database.Password,
//		cfg.Database.Host,
//		cfg.Database.Port,
//		cfg.Database.DatabaseName,
//	)
//
//	log.Printf("Connecting to MySQL: %s:%s/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.DatabaseName)
//
//	/*
//	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
//	if err != nil {
//		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
//	}
//	dbmysql.SetDB(db)
//
//	db, err := dbmysql.kkkjj
//
//	if err := db.AutoMigrate(
//		&dbmysql.Notification{},
//		&dbmysql.Device{},
//	); err != nil {
//		log.Printf("Migration warning: %v", err)
//	}
//
//	return db, nil
//}
//*/
//
//func ProvideFirebaseApp(cfg *config.Config) (*firebase.App, error) {
//	if !cfg.Firebase.Enabled || cfg.Firebase.CredentialsFilePath == "" {
//		log.Println("Firebase disabled or credentials missing")
//		return nil, nil
//	}
//
//	opt := option.WithCredentialsFile(cfg.Firebase.CredentialsFilePath)
//	firebaseConfig := &firebase.Config{ProjectID: cfg.Firebase.ProjectID}
//
//	app, err := firebase.NewApp(context.Background(), firebaseConfig, opt)
//	if err != nil {
//		log.Printf("Firebase init error: %v", err)
//		return nil, nil
//	}
//
//	return app, nil
//}
//
//func ProvideFirebaseMessaging(app *firebase.App) (*messaging.Client, error) {
//	if app == nil {
//		log.Println("No Firebase app provided")
//		return nil, nil
//	}
//
//	client, err := app.Messaging(context.Background())
//	if err != nil {
//		log.Printf("Failed to get FCM client: %v", err)
//		return nil, nil
//	}
//	return client, nil
//}
//
//func ProvideEmailService(cfg *config.Config) common.EmailService {
//	return &MockEmailService{}
//}
//
//type MockEmailService struct{}
//
//func (m *MockEmailService) SendEmail(to, subject, body string) error {
//	log.Printf("Mock Email - To: %s | Subject: %s", to, subject)
//	return nil
//}
