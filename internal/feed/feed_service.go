package feed

import (
	"context"
	"time"

	"GoSocial/internal/dbmysql"
)

// type FeedService struct {
// 	repo *FeedRepository
// }

// func NewFeedService(repo *FeedRepository) *FeedService {
// 	return &FeedService{repo: repo}
// }

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

func (s *FeedService) CreateContent(ctx context.Context, content *dbmysql.Content) (int64, error) {
	content.CreatedAt = time.Now()
	content.UpdatedAt = time.Now()
	err := s.contentRepo.CreateContent(ctx, content)
	return content.ContentID, err
}

func (s *FeedService) GetContent(ctx context.Context, id int64) (*dbmysql.Content, error) {
	return s.contentRepo.GetContentByID(ctx, id)
}

func (s *FeedService) ListUserContent(ctx context.Context, userID int64) ([]dbmysql.Content, error) {
	return s.contentRepo.ListUserContent(ctx, userID)
}

func (s *FeedService) DeleteContent(ctx context.Context, id int64) error {
	return s.contentRepo.DeleteContent(ctx, id)
}

//
// --------- MEDIA REF ---------
//

func (s *FeedService) CreateMediaRef(ctx context.Context, media *dbmysql.MediaRef) (int64, error) {
	media.UploadedAt = time.Now()
	err := s.mediaRepo.CreateMediaRef(ctx, media)
	return media.MediaRefID, err
}

func (s *FeedService) GetMediaRef(ctx context.Context, id int64) (*dbmysql.MediaRef, error) {
	return s.mediaRepo.GetMediaRefByID(ctx, id)
}

//
// --------- REACTIONS ---------
//

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
