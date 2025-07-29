package feed

import (
	"context"
	"fmt"
	"time"

	"GoSocial/internal/dbmysql"
)

var MediaBaseURL = "http://localhost:8080/media/"

func GetMediaURL(fileName string) string {
	return fmt.Sprintf("%s%s", MediaBaseURL, fileName)
}

type FeedUsecase interface {
	CreateContent(ctx context.Context, content *dbmysql.Content) (int64, error)
	GetContent(ctx context.Context, id int64) (*dbmysql.Content, error)
	ListUserContent(ctx context.Context, userID int64) ([]dbmysql.Content, error)
	DeleteContent(ctx context.Context, id int64) error

	CreateMediaRef(ctx context.Context, media *dbmysql.MediaRef) (int64, error)
	GetMediaRef(ctx context.Context, id int64) (*dbmysql.MediaRef, error)

	AddReaction(ctx context.Context, reaction *dbmysql.Reaction) error
	GetReactions(ctx context.Context, contentID int64) ([]dbmysql.Reaction, error)
	DeleteReaction(ctx context.Context, userID, contentID int64) error
}

type FeedService struct {
	contentRepo  Content
	mediaRepo    MediaRef
	reactionRepo Reactions
}

func NewFeedService(c Content, m MediaRef, r Reactions) *FeedService {
	return &FeedService{
		contentRepo:  c,
		mediaRepo:    m,
		reactionRepo: r,
	}
}

// --------- CONTENT ---------

// CreateContent creates new content and uploads media if provided.

func (s *FeedService) CreateContent(ctx context.Context, content *dbmysql.Content, fileData []byte, mediatype string, medianame string) (int64, error) {
	content.CreatedAt = time.Now()
	content.UpdatedAt = time.Now()

	// Step 1: Upload media only if file is passed
	if fileData != nil && len(fileData) > 0 {
		// Create a temporary MediaRef struct with required fields
		media := &dbmysql.MediaRef{
			Type:       mediatype, // or infer from file
			FileName:   medianame, // maybe from HTTP request
			UploadedBy: content.AuthorID,
		}

		// Upload media and persist
		if err := s.mediaRepo.CreateMediaRef(ctx, media, fileData); err != nil {
			return 0, err
		}

		// Link saved media_ref_id back to content
		content.MediaRefID = &media.MediaRefID
	}

	// Step 2: Save content to MySQL
	if err := s.contentRepo.CreateContent(ctx, content); err != nil {
		return 0, err
	}

	return content.ContentID, nil
}

// func (s *FeedService) GetContent(ctx context.Context, id int64) (*dbmysql.Content, error) {
// 	return s.contentRepo.GetContentByID(ctx, id)
// }

func (s *FeedService) GetContent(ctx context.Context, id int64) (*dbmysql.Content, string, error) {
	// Step 1: Get content
	content, err := s.contentRepo.GetContentByID(ctx, id)
	if err != nil {
		return nil, "", err
	}

	// Step 2: Get media URL (if present)
	var mediaURL string
	if content.MediaRefID != nil {
		mediaMeta, _, err := s.mediaRepo.GetMediaRefByID(ctx, *content.MediaRefID)
		if err == nil {
			mediaURL = GetMediaURL(mediaMeta.FilePath) // FilePath = GridFS ObjectID
		}
	}

	// Step 3: Return content + optional media URL
	return content, mediaURL, nil
}

func (s *FeedService) ListUserContent(ctx context.Context, userID int64) ([]dbmysql.Content, error) {
	return s.contentRepo.ListUserContent(ctx, userID)
}

// DeleteContent deletes content and its associated media if present.

func (s *FeedService) DeleteContent(ctx context.Context, id int64) error {
	// Step 1: Load content to get media_ref_id
	content, err := s.contentRepo.GetContentByID(ctx, id)
	if err != nil {
		return err
	}

	// Step 2: Delete associated media if present
	if content.MediaRefID != nil {
		_ = s.mediaRepo.DeleteMedia(ctx, *content.MediaRefID) // Don't fail content delete if this fails
	}

	// Step 3: Delete content
	return s.contentRepo.DeleteContent(ctx, id)
}

// --------- MEDIA REF ---------

func (s *FeedService) CreateMediaRef(ctx context.Context, media *dbmysql.MediaRef, fileData []byte) (int64, error) {
	media.UploadedAt = time.Now()
	err := s.mediaRepo.CreateMediaRef(ctx, media, fileData)
	return media.MediaRefID, err
}

func (s *FeedService) GetMediaRef(ctx context.Context, id int64) (*dbmysql.MediaRef, []byte, error) {
	return s.mediaRepo.GetMediaRefByID(ctx, id)
}

// --------- REACTIONS ---------

func (s *FeedService) AddReaction(ctx context.Context, reaction *dbmysql.Reaction) error {
	reaction.CreatedAt = time.Now()
	return s.reactionRepo.AddReaction(ctx, reaction)
}

func (s *FeedService) GetReactions(ctx context.Context, contentID int64) ([]dbmysql.Reaction, error) {
	return s.reactionRepo.GetReactionsForContent(ctx, contentID)
}

func (s *FeedService) DeleteReaction(ctx context.Context, userID, contentID int64) error {
	return s.reactionRepo.DeleteReaction(ctx, userID, contentID)
}
