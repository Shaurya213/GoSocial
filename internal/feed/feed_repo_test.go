package feed

import (
	"context"
	"testing"
	"time"

	//"GoSocial/internal/db"
	"GoSocial/internal/dbmysql"
)

func TestCreateAndGetContent(t *testing.T) {
	conn, err := db.NewMySQLConnection("root", "root", "gosocial_test")
	if err != nil {
		t.Fatalf("DB connection failed: %v", err)
	}

	conn.AutoMigrate(&dbmysql.Content{}) // Create table if not exists

	repo := NewFeedRepository(conn)

	content := &dbmysql.Content{
		AuthorID:    1,
		Type:        "POST",
		TextContent: ptr("Hello Test"),
		Privacy:     "public",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err = repo.CreateContent(context.Background(), content)
	if err != nil {
		t.Fatalf("Failed to create content: %v", err)
	}

	got, err := repo.GetContentByID(context.Background(), content.ContentID)
	if err != nil {
		t.Fatalf("Failed to get content: %v", err)
	}

	if got.AuthorID != content.AuthorID || got.Type != content.Type {
		t.Errorf("Expected %+v, got %+v", content, got)
	}
}

func ptr(s string) *string {
	return &s
}
