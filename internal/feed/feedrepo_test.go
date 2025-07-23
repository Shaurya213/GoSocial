package feed

import (
	"context"
	"fmt"
	"testing"
	"time"

	"GoSocial/internal/dbmysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewMySQLConnection(user, password, dbname string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(localhost:3306)/%s?parseTime=true", user, password, dbname)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}

func ptr(s string) *string {
	return &s
}

func TestCreateAndGetContent(t *testing.T) {
	db, err := NewMySQLConnection("root", "root", "gosocial_test")
	if err != nil {
		t.Fatalf("failed to connect to DB: %v", err)
	}
	_ = db.AutoMigrate(&dbmysql.Content{}, &dbmysql.MediaRef{}, &dbmysql.Reaction{})
	repo := NewFeedRepository(db)

	now := time.Now()
	content := &dbmysql.Content{
		AuthorID:    1,
		Type:        "POST",
		TextContent: ptr("This is a test post"),
		Privacy:     "public",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err = repo.CreateContent(context.Background(), content)
	if err != nil {
		t.Fatalf("CreateContent failed: %v", err)
	}

	got, err := repo.GetContentByID(context.Background(), content.ContentID)
	if err != nil {
		t.Fatalf("GetContentByID failed: %v", err)
	}
	if got.AuthorID != content.AuthorID || got.TextContent == nil || *got.TextContent != *content.TextContent {
		t.Errorf("Expected %+v, got %+v", content, got)
	}
}

func TestListUserContent(t *testing.T) {
	db, _ := NewMySQLConnection("root", "root", "gosocial_test")
	repo := NewFeedRepository(db)
	ctx := context.Background()

	contents, err := repo.ListUserContent(ctx, 1)
	if err != nil {
		t.Fatalf("ListUserContent failed: %v", err)
	}
	if len(contents) == 0 {
		t.Log("No user content yet. Might be empty.")
	}
}

func TestDeleteContent(t *testing.T) {
	db, _ := NewMySQLConnection("root", "root", "gosocial_test")
	repo := NewFeedRepository(db)
	ctx := context.Background()

	content := &dbmysql.Content{
		AuthorID:    1,
		Type:        "POST",
		TextContent: ptr("Delete this"),
		Privacy:     "public",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	_ = repo.CreateContent(ctx, content)

	err := repo.DeleteContent(ctx, content.ContentID)
	if err != nil {
		t.Fatalf("DeleteContent failed: %v", err)
	}
}

func TestCreateAndGetMediaRef(t *testing.T) {
	db, _ := NewMySQLConnection("root", "root", "gosocial_test")
	repo := NewFeedRepository(db)
	ctx := context.Background()

	media := &dbmysql.MediaRef{
		Type:       "image",
		FilePath:   "/media/test.png",
		FileName:   "test.png",
		UploadedBy: 1,
		UploadedAt: time.Now(),
		SizeBytes:  1024,
	}
	err := repo.CreateMediaRef(ctx, media)
	if err != nil {
		t.Fatalf("CreateMediaRef failed: %v", err)
	}

	got, err := repo.GetMediaRefByID(ctx, media.MediaRefID)
	if err != nil {
		t.Fatalf("GetMediaRefByID failed: %v", err)
	}
	if got.FileName != media.FileName {
		t.Errorf("Expected %s, got %s", media.FileName, got.FileName)
	}
}

func TestAddGetDeleteReaction(t *testing.T) {
	db, _ := NewMySQLConnection("root", "root", "gosocial_test")
	repo := NewFeedRepository(db)
	ctx := context.Background()

	reaction := &dbmysql.Reaction{
		UserID:    1,
		ContentID: 1,
		Type:      "like",
		CreatedAt: time.Now(),
	}

	err := repo.AddReaction(ctx, reaction)
	if err != nil {
		t.Fatalf("AddReaction failed: %v", err)
	}

	list, err := repo.GetReactionsForContent(ctx, 1)
	if err != nil {
		t.Fatalf("GetReactionsForContent failed: %v", err)
	}
	if len(list) == 0 {
		t.Errorf("Expected at least one reaction")
	}

	err = repo.DeleteReaction(ctx, 1, 1)
	if err != nil {
		t.Fatalf("DeleteReaction failed: %v", err)
	}
}
