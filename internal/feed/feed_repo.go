package feed

import (
	"GoSocial/internal/dbmysql"
	"context"
	"gorm.io/gorm"
)

type FeedRepository struct {
	db *gorm.DB
}

func NewFeedRepository(db *gorm.DB) *FeedRepository {
	return &FeedRepository{db: db}
}

//
// --------- CONTENT ---------
//

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

//
// --------- MEDIA REF ---------
//

func (r *FeedRepository) CreateMediaRef(ctx context.Context, media *dbmysql.MediaRef) error {
	return r.db.WithContext(ctx).Create(media).Error
}

func (r *FeedRepository) GetMediaRefByID(ctx context.Context, id int64) (*dbmysql.MediaRef, error) {
	var media dbmysql.MediaRef
	err := r.db.WithContext(ctx).First(&media, "media_ref_id = ?", id).Error
	return &media, err
}

//
// --------- REACTIONS ---------
//

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
