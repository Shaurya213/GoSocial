package feed

import (
	"context"
	"fmt"
	"sort"
	"time"

	userpb "GoSocial/api/v1/user"
	"GoSocial/internal/dbmysql"
)

// media helper to generate media URLs
var MediaBaseURL = "http://localhost:8080/media/"

func GetMediaURL(fileName string) string {
	return fmt.Sprintf("%s%s", MediaBaseURL, fileName)
}

// all functions in this file are higher-order functions that call the core service methods
type FeedUsecase interface {
	CreatePost(ctx context.Context, authorID int64, text string, fileData []byte, fileName string, mediaType string, privacy string) (int64, error)
	CreateReel(ctx context.Context, authorID int64, caption string, fileData []byte, fileName string, durationSecs int, privacy string) (int64, error)
	CreateStory(ctx context.Context, authorID int64, fileData []byte, mediaType string, mediaName string, durationSec int, privacy string) (int64, error)
	ReactToContent(ctx context.Context, userID, contentID int64, reactionType string) error
	GetReactions(ctx context.Context, contentID int64) ([]dbmysql.Reaction, error)
	DeleteReaction(ctx context.Context, userID, contentID int64) error
	GetTimeline(ctx context.Context, userID int64) ([]dbmysql.Content, []string, error)
	GetUserContent(ctx context.Context, requesterID, targetUserID int64) ([]dbmysql.Content, []string, error)

	GetMediaRef(ctx context.Context, id int64) (*dbmysql.MediaRef, error)

	GetContent(ctx context.Context, id int64) (*dbmysql.Content, string, error)
	DeleteContent(ctx context.Context, id int64) error
}

type FeedService struct {
	contentRepo    Content
	mediaRepo      MediaRef
	reactionRepo   Reactions
	UserClient     userpb.UserServiceClient
	cleanupStarted bool
}

func NewFeedService(c Content, m MediaRef, r Reactions, u userpb.UserServiceClient) *FeedService {
	service := &FeedService{
		contentRepo:  c,
		mediaRepo:    m,
		reactionRepo: r,
		UserClient:   u,
	}
	go service.startExpiredStoryCleaner()

	return service
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

func (s *FeedService) GetUserFriendIDs(ctx context.Context, userID int64) ([]int64, error) {
	resp, err := s.UserClient.ListFriends(ctx, &userpb.UserID{UserId: userID})
	if err != nil {
		return nil, fmt.Errorf("ListFriends failed: %w", err)
	}

	friendIDs := make([]int64, 0, len(resp.Friends))
	for _, f := range resp.Friends {
		friendIDs = append(friendIDs, f.UserId)
	}
	return friendIDs, nil
}

func (s *FeedService) startExpiredStoryCleaner() {
	if s.cleanupStarted {
		return
	}
	s.cleanupStarted = true

	ticker := time.NewTicker(10 * time.Minute) // ⏱️ runs every 10 minutes
	for {
		<-ticker.C

		go func() {
			now := time.Now()
			expired, err := s.contentRepo.ListExpiredStories(context.Background(), now)
			if err != nil {
				fmt.Println("Failed to fetch expired stories:", err)
				return
			}

			for _, story := range expired {
				if err := s.DeleteContent(context.Background(), story.ContentID); err != nil {
					fmt.Printf("Failed to delete expired story %d: %v\n", story.ContentID, err)
				}
			}
		}()
	}
}

// -----------higher-order functions-----------

func (s *FeedService) CreatePost(
	ctx context.Context,
	authorID int64,
	text string,
	fileData []byte,
	fileName string,
	mediaType string,
	privacy string,
) (int64, error) {

	content := &dbmysql.Content{
		AuthorID:    authorID,
		Type:        "POST",
		TextContent: &text,
		Privacy:     privacy,
	}

	// Just reuse the existing core service logic
	return s.CreateContent(ctx, content, fileData, mediaType, fileName)
}

func (s *FeedService) CreateReel(
	ctx context.Context,
	authorID int64,
	caption string,
	fileData []byte,
	fileName string,
	durationSecs int,
	privacy string,
) (int64, error) {

	// Validate inputs if needed (e.g., ensure fileData is a video)

	content := &dbmysql.Content{
		AuthorID:    authorID,
		Type:        "REEL",
		TextContent: &caption,
		Privacy:     privacy,
		Duration:    &durationSecs,
	}

	return s.CreateContent(ctx, content, fileData, "video", fileName)
}

func (s *FeedService) CreateStory(
	ctx context.Context,
	authorID int64,
	fileData []byte,
	mediaType string,
	mediaName string,
	durationSec int,
	privacy string,
) (int64, error) {
	content := &dbmysql.Content{
		AuthorID: authorID,
		Type:     "STORY",
		Privacy:  privacy,
	}
	duration := durationSec
	content.Duration = &duration
	expiration := content.CreatedAt.Add(time.Duration(durationSec) * time.Second)
	content.Expiration = &expiration

	// Call common content creation logic
	return s.CreateContent(ctx, content, fileData, mediaType, mediaName)
}

func (s *FeedService) ReactToContent(ctx context.Context, userID, contentID int64, reactionType string) error {
	// Step 1: Delete existing reaction if any
	_ = s.DeleteReaction(ctx, userID, contentID) // Ignore error if not found

	// Step 2: Add new reaction
	reaction := &dbmysql.Reaction{
		UserID:    userID,
		ContentID: contentID,
		Type:      reactionType,
		CreatedAt: time.Now(),
	}
	return s.AddReaction(ctx, reaction)
}

func (s *FeedService) GetTimeline(ctx context.Context, userID int64) ([]dbmysql.Content, []string, error) {
	// Step 1: Get friend IDs
	friendResp, err := s.UserClient.ListFriends(ctx, &userpb.UserID{UserId: userID})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch friends: %w", err)
	}

	userIDs := []int64{userID}
	for _, f := range friendResp.Friends {
		userIDs = append(userIDs, f.UserId)
	}

	// Step 2: Collect content
	var allContent []dbmysql.Content
	var allURLs []string

	for _, uid := range userIDs {
		contentList, err := s.contentRepo.ListUserContent(ctx, uid)
		if err != nil {
			continue
		}
		for _, c := range contentList {
			allContent = append(allContent, c)

			if c.MediaRefID != nil {
				mediaMeta, _, err := s.mediaRepo.GetMediaRefByID(ctx, *c.MediaRefID)
				if err == nil {
					allURLs = append(allURLs, GetMediaURL(mediaMeta.FilePath))
				} else {
					allURLs = append(allURLs, "")
				}
			} else {
				allURLs = append(allURLs, "")
			}
		}
	}

	// Step 3: Sort
	sort.SliceStable(allContent, func(i, j int) bool {
		return allContent[i].CreatedAt.After(allContent[j].CreatedAt)
	})

	return allContent, allURLs, nil
}

func (s *FeedService) GetUserContent(ctx context.Context, requesterID, targetUserID int64) ([]dbmysql.Content, []string, error) {
	// Step 1: Fetch all content
	allContent, err := s.contentRepo.ListUserContent(ctx, targetUserID)
	if err != nil {
		return nil, nil, err
	}

	// Step 2: Self view
	if requesterID == targetUserID {
		var urls []string
		for _, c := range allContent {
			if c.MediaRefID != nil {
				mediaMeta, _, err := s.mediaRepo.GetMediaRefByID(ctx, *c.MediaRefID)
				if err == nil {
					urls = append(urls, GetMediaURL(mediaMeta.FilePath))
				} else {
					urls = append(urls, "")
				}
			} else {
				urls = append(urls, "")
			}
		}
		return allContent, urls, nil
	}

	// Step 3: Friendship check
	friendResp, err := s.UserClient.ListFriends(ctx, &userpb.UserID{UserId: targetUserID})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch friends: %w", err)
	}

	isFriend := false
	for _, f := range friendResp.Friends {
		if f.UserId == requesterID {
			isFriend = true
			break
		}
	}

	// Step 4: Filter + media URLs
	var filtered []dbmysql.Content
	var mediaURLs []string

	for _, c := range allContent {
		if c.Privacy == "public" || (c.Privacy == "friends" && isFriend) {
			filtered = append(filtered, c)

			if c.MediaRefID != nil {
				mediaMeta, _, err := s.mediaRepo.GetMediaRefByID(ctx, *c.MediaRefID)
				if err == nil {
					mediaURLs = append(mediaURLs, GetMediaURL(mediaMeta.FilePath))
				} else {
					mediaURLs = append(mediaURLs, "")
				}
			} else {
				mediaURLs = append(mediaURLs, "")
			}
		}
	}

	return filtered, mediaURLs, nil
}
