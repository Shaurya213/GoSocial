package feed

import (
	"bytes"
	"context"
	"fmt"
	"gorm.io/gorm"
	"gosocial/internal/dbmongo"
	"gosocial/internal/dbmysql"
	"io"
	"time"
)

type FeedRepository struct {
	gridClient *dbmongo.MediaStorage
	db         *gorm.DB
}

func NewFeedRepository(db *gorm.DB, gridClient *dbmongo.MediaStorage) *FeedRepository {
	return &FeedRepository{db: db, gridClient: gridClient}
}

// --------- CONTENT ---------
type Content interface {
	CreateContent(ctx context.Context, content *dbmysql.Content) error
	GetContentByID(ctx context.Context, id int64) (*dbmysql.Content, error)
	ListUserContent(ctx context.Context, userID int64) ([]dbmysql.Content, error)
	DeleteContent(ctx context.Context, id int64) error
	ListExpiredStories(ctx context.Context, now time.Time) ([]dbmysql.Content, error)
}

func (r *FeedRepository) CreateContent(ctx context.Context, content *dbmysql.Content) error {
	return r.db.WithContext(ctx).Create(content).Error
}

func (r *FeedRepository) GetContentByID(ctx context.Context, id int64) (*dbmysql.Content, error) {
	var content dbmysql.Content
	err := r.db.WithContext(ctx).First(&content, "content_id = ?", id).Error
	return &content, err
}

func (r *FeedRepository) ListUserContent(ctx context.Context, userID int64) ([]dbmysql.Content, error) {
	var contents []dbmysql.Content
	err := r.db.WithContext(ctx).
		Where("author_id = ?", userID).
		Order("created_at DESC").
		Find(&contents).Error
	return contents, err
}

func (r *FeedRepository) DeleteContent(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&dbmysql.Content{}, "content_id = ?", id).Error
}

// --------- MEDIA REF ---------
type MediaRef interface {
	CreateMediaRef(ctx context.Context, media *dbmysql.MediaRef, fileData []byte) error
	GetMediaRefByID(ctx context.Context, id int64) (*dbmysql.MediaRef, []byte, error)
	DeleteMedia(ctx context.Context, mediaRefID int64) error
}

func (r *FeedRepository) CreateMediaRef(ctx context.Context, media *dbmysql.MediaRef, fileData []byte) error {
	// Step 1: Upload file to GridFS
	mediaFile, err := r.gridClient.UploadFile(ctx, media.FileName, "application/octet-stream", fmt.Sprint(media.UploadedBy), bytes.NewReader(fileData))
	if err != nil {
		return err
	}

	// Step 2: Store fileID in SQL media_ref.FileID (updated field name)
	media.FileID = mediaFile.ID // GridFS ID as string
	media.UploadedAt = mediaFile.UploadedAt
	media.Size = mediaFile.Size

	// Step 3: Save metadata to MySQL
	return r.db.WithContext(ctx).Create(media).Error
}

func (r *FeedRepository) GetMediaRefByID(ctx context.Context, id int64) (*dbmysql.MediaRef, []byte, error) {
	var media dbmysql.MediaRef

	// Step 1: Get media metadata from SQL
	err := r.db.WithContext(ctx).First(&media, "media_ref_id = ?", id).Error
	if err != nil {
		return nil, nil, err
	}

	// Step 2: Use updated DownloadFile from media_storage.go
	reader, _, err := r.gridClient.DownloadFile(ctx, media.FileID)
	if err != nil {
		return nil, nil, err
	}

	// Step 3: Read file content
	fileData, err := io.ReadAll(reader)
	if err != nil {
		return nil, nil, err
	}

	return &media, fileData, nil
}

func (r *FeedRepository) DeleteMedia(ctx context.Context, mediaRefID int64) error {
	// Step 1: Get file path from SQL
	var media dbmysql.MediaRef
	err := r.db.WithContext(ctx).First(&media, "media_ref_id = ?", mediaRefID).Error
	if err != nil {
		return err
	}

	// Step 2: Delete from GridFS
	if err := r.gridClient.DeleteFile(ctx, media.FileID); err != nil {
		return err
	}

	// Step 3: Delete metadata
	return r.db.WithContext(ctx).Delete(&media).Error
}

// --------- REACTIONS ---------
type Reactions interface {
	AddReaction(ctx context.Context, reaction *dbmysql.Reaction) error
	GetReactionsForContent(ctx context.Context, contentID int64) ([]dbmysql.Reaction, error)
	DeleteReaction(ctx context.Context, userID, contentID int64) error
}

func (r *FeedRepository) AddReaction(ctx context.Context, reaction *dbmysql.Reaction) error {
	return r.db.WithContext(ctx).Create(reaction).Error
}

func (r *FeedRepository) GetReactionsForContent(ctx context.Context, contentID int64) ([]dbmysql.Reaction, error) {
	var reactions []dbmysql.Reaction
	err := r.db.WithContext(ctx).
		Where("content_id = ?", contentID).
		Find(&reactions).Error
	return reactions, err
}

func (r *FeedRepository) DeleteReaction(ctx context.Context, userID, contentID int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND content_id = ?", userID, contentID).
		Delete(&dbmysql.Reaction{}).Error
}

func (r *FeedRepository) ListExpiredStories(ctx context.Context, now time.Time) ([]dbmysql.Content, error) {
	var stories []dbmysql.Content
	err := r.db.WithContext(ctx).
		Where("type = ? AND expiration IS NOT NULL AND expiration <= ?", "STORY", now).
		Find(&stories).Error
	return stories, err
}
