package feed

import (
	"GoSocial/internal/dbmongo"
	"GoSocial/internal/dbmysql"
	"context"
	"gorm.io/gorm"
	"time"
)

type FeedRepository struct {
	gridClient *dbmongo.GridFSClient
	db         *gorm.DB
}

func NewFeedRepository(db *gorm.DB, gridclient *dbmongo.GridFSClient) *FeedRepository {
	return &FeedRepository{db: db, gridClient: gridclient}
}

// --------- CONTENT ---------
type Content interface {
	CreateContent(ctx context.Context, content *dbmysql.Content) error
	GetContentByID(ctx context.Context, id int64) (*dbmysql.Content, error)
	ListUserContent(ctx context.Context, userID int64) ([]dbmysql.Content, error)
	DeleteContent(ctx context.Context, id int64) error
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
	CreateMediaRef(ctx context.Context, media *dbmysql.MediaRef) error
	GetMediaRefByID(ctx context.Context, id int64) (*dbmysql.MediaRef, error)
}

func (r *FeedRepository) CreateMediaRef(ctx context.Context, media *dbmysql.MediaRef, fileData []byte) error {
	// Step 1: Upload file to GridFS
	fileID, err := r.gridClient.UploadFile(ctx, media.FileName, fileData)
	if err != nil {
		return err
	}

	// Step 2: Store fileID in SQL media_ref.file_path
	media.FilePath = fileID.Hex() // Assuming fileID is an ObjectID and we store its hex representation
	media.UploadedAt = time.Now()

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

	// Step 2: Get actual file content from Mongo GridFS using file path (ObjectID)
	fileData, err := r.gridClient.GetFileByID(ctx, media.FilePath)
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
	if err := r.gridClient.DeleteFile(ctx, media.FilePath); err != nil {
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
