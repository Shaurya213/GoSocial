package feed

import (
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"gosocial/internal/dbmongo"
	"gosocial/internal/dbmysql"
	"strconv"

	"context"

	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	db         *gorm.DB
	gridClient *dbmongo.MediaStorage
	repo       *FeedRepository
)

func setup(t *testing.T) {
	t.Helper()

	// --- MySQL ---
	dsn := "gosocial_user:G0Social@123@tcp(localhost:3306)/gosocial_test?parseTime=true&multiStatements=true"
	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		// Critical for tests: avoid GORM generating FK constraints that flip directions
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("Failed to connect to MySQL: %v", err)
	}

	// Migrate in safe order; include users because media_ref has UploadedBy -> users.user_id
	if err := db.AutoMigrate(
		&dbmysql.User{},
		&dbmysql.Content{},
		&dbmysql.MediaRef{},
		&dbmysql.Reaction{},
	); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	// --- MongoDB (with auth, matches docker-compose) ---
	mongoURI := "mongodb://admin:admin123@localhost:27017/?authSource=admin"
	mc, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	mdb := mc.Database("gosocial_test")

	bucket, err := gridfs.NewBucket(mdb)
	if err != nil {
		t.Fatalf("Failed to create GridFS bucket: %v", err)
	}

	// Build the wrapper expected by NewMediaStorage
	mongoClient := &dbmongo.MongoClient{
		Database: mdb,
		GridFS:   bucket,
	}

	gridClient = dbmongo.NewMediaStorage(mongoClient)

	// Wire the repository (use the GORM `db`, not `mdb`)
	repo = NewFeedRepository(db, gridClient)
}

func ptr(s string) *string {
	return &s
}

func TestContentCRUD(t *testing.T) {
	setup(t)
	now := time.Now()
	content := &dbmysql.Content{
		AuthorID:    1,
		Type:        "POST",
		TextContent: ptr("Sample post"),
		Privacy:     "public",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := repo.CreateContent(context.Background(), content)
	if err != nil {
		t.Fatalf("CreateContent failed: %v", err)
	}

	got, err := repo.GetContentByID(context.Background(), content.ContentID)
	if err != nil {
		t.Fatalf("GetContentByID failed: %v", err)
	}
	if got.TextContent == nil || *got.TextContent != *content.TextContent {
		t.Errorf("TextContent mismatch")
	}

	list, err := repo.ListUserContent(context.Background(), content.AuthorID)
	if err != nil {
		t.Fatalf("ListUserContent failed: %v", err)
	}
	if len(list) == 0 {
		t.Error("Expected at least one content")
	}

	err = repo.DeleteContent(context.Background(), content.ContentID)
	if err != nil {
		t.Fatalf("DeleteContent failed: %v", err)
	}
}

func TestMediaRefCRUD(t *testing.T) {
	setup(t)
	media := &dbmysql.MediaRef{
		Type:       "image",
		FileName:   "test.png",
		UploadedBy: strconv.Itoa(1),
		UploadedAt: time.Now(),
	}
	data := []byte("sample content")

	err := repo.CreateMediaRef(context.Background(), media, data)
	if err != nil {
		t.Fatalf("CreateMediaRef failed: %v", err)
	}

	meta, fetchedData, err := repo.GetMediaRefByID(context.Background(), int64(media.MediaRefID))
	if err != nil {
		t.Fatalf("GetMediaRefByID failed: %v", err)
	}
	if meta.FileName != media.FileName || string(fetchedData) != string(data) {
		t.Errorf("Mismatch in media data")
	}

	err = repo.DeleteMedia(context.Background(), int64(media.MediaRefID))
	if err != nil {
		t.Fatalf("DeleteMedia failed: %v", err)
	}
}

func TestReactionFlow(t *testing.T) {
	setup(t)
	reaction := &dbmysql.Reaction{
		UserID:    2,
		ContentID: 1,
		Type:      "love",
		CreatedAt: time.Now(),
	}

	err := repo.AddReaction(context.Background(), reaction)
	if err != nil {
		t.Fatalf("AddReaction failed: %v", err)
	}

	list, err := repo.GetReactionsForContent(context.Background(), reaction.ContentID)
	if err != nil {
		t.Fatalf("GetReactionsForContent failed: %v", err)
	}
	if len(list) == 0 {
		t.Error("Expected reaction")
	}

	err = repo.DeleteReaction(context.Background(), reaction.UserID, reaction.ContentID)
	if err != nil {
		t.Fatalf("DeleteReaction failed: %v", err)
	}
}

func TestListExpiredStories(t *testing.T) {
	setup(t)
	expired := time.Now().Add(-1 * time.Hour)
	content := &dbmysql.Content{
		AuthorID:    3,
		Type:        "STORY",
		TextContent: ptr("expired story"),
		Privacy:     "public",
		Expiration:  &expired,
		CreatedAt:   time.Now().Add(-2 * time.Hour),
		UpdatedAt:   time.Now().Add(-2 * time.Hour),
	}

	err := repo.CreateContent(context.Background(), content)
	if err != nil {
		t.Fatalf("CreateContent for story failed: %v", err)
	}

	list, err := repo.ListExpiredStories(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("ListExpiredStories failed: %v", err)
	}
	if len(list) == 0 {
		t.Error("Expected expired stories, got none")
	}
}
