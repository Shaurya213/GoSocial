package feed

import (
	"context"
	"errors"
	"testing"

	feedpb "GoSocial/api/v1/feed"
	"GoSocial/internal/dbmysql"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ---- Fake FeedUsecase for handler tests ----

type fakeFeedSvc struct {
	CreatePostFn     func(ctx context.Context, authorID int64, text string, fileData []byte, fileName, mediaType, privacy string) (int64, error)
	CreateReelFn     func(ctx context.Context, authorID int64, caption string, fileData []byte, fileName string, durationSecs int, privacy string) (int64, error)
	CreateStoryFn    func(ctx context.Context, authorID int64, fileData []byte, mediaType, mediaName string, durationSec int, privacy string) (int64, error)
	ReactToContentFn func(ctx context.Context, userID, contentID int64, reactionType string) error
	GetReactionsFn   func(ctx context.Context, contentID int64) ([]dbmysql.Reaction, error)
	DeleteReactionFn func(ctx context.Context, userID, contentID int64) error
	GetTimelineFn    func(ctx context.Context, userID int64) ([]dbmysql.Content, []string, error)
	GetUserContentFn func(ctx context.Context, requesterID, targetUserID int64) ([]dbmysql.Content, []string, error)
	GetMediaRefFn    func(ctx context.Context, id int64) (*dbmysql.MediaRef, error)
	GetContentFn     func(ctx context.Context, id int64) (*dbmysql.Content, string, error)
	DeleteContentFn  func(ctx context.Context, id int64) error
}

func (f *fakeFeedSvc) CreatePost(ctx context.Context, a int64, t string, d []byte, n, mt, p string) (int64, error) {
	return f.CreatePostFn(ctx, a, t, d, n, mt, p)
}
func (f *fakeFeedSvc) CreateReel(ctx context.Context, a int64, c string, d []byte, n string, dur int, p string) (int64, error) {
	return f.CreateReelFn(ctx, a, c, d, n, dur, p)
}
func (f *fakeFeedSvc) CreateStory(ctx context.Context, a int64, d []byte, mt, mn string, dur int, p string) (int64, error) {
	return f.CreateStoryFn(ctx, a, d, mt, mn, dur, p)
}
func (f *fakeFeedSvc) ReactToContent(ctx context.Context, u, c int64, r string) error {
	return f.ReactToContentFn(ctx, u, c, r)
}
func (f *fakeFeedSvc) GetReactions(ctx context.Context, c int64) ([]dbmysql.Reaction, error) {
	return f.GetReactionsFn(ctx, c)
}
func (f *fakeFeedSvc) DeleteReaction(ctx context.Context, u, c int64) error {
	return f.DeleteReactionFn(ctx, u, c)
}
func (f *fakeFeedSvc) GetTimeline(ctx context.Context, u int64) ([]dbmysql.Content, []string, error) {
	return f.GetTimelineFn(ctx, u)
}
func (f *fakeFeedSvc) GetUserContent(ctx context.Context, r, t int64) ([]dbmysql.Content, []string, error) {
	return f.GetUserContentFn(ctx, r, t)
}
func (f *fakeFeedSvc) GetMediaRef(ctx context.Context, id int64) (*dbmysql.MediaRef, error) {
	return f.GetMediaRefFn(ctx, id)
}
func (f *fakeFeedSvc) GetContent(ctx context.Context, id int64) (*dbmysql.Content, string, error) {
	return f.GetContentFn(ctx, id)
}
func (f *fakeFeedSvc) DeleteContent(ctx context.Context, id int64) error {
	return f.DeleteContentFn(ctx, id)
}

func newHandlerWithFake(f *fakeFeedSvc) *FeedHandlers {
	return &FeedHandlers{FeedSvc: f}
}

// ---- Tests ----

func TestCreatePost_ValidationErrors(t *testing.T) {
	h := newHandlerWithFake(&fakeFeedSvc{})

	// invalid author id
	_, err := h.CreatePost(context.Background(), &feedpb.CreatePostRequest{
		AuthorId: 0, Text: "hi", MediaType: "image", Privacy: "public", MediaName: "x.jpg",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for author id, got %v", err)
	}

	// no text and no media
	_, err = h.CreatePost(context.Background(), &feedpb.CreatePostRequest{
		AuthorId: 1, Text: "", MediaType: "image", Privacy: "public", MediaName: "x.jpg",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for empty content, got %v", err)
	}

	// missing media type
	_, err = h.CreatePost(context.Background(), &feedpb.CreatePostRequest{
		AuthorId: 1, Text: "hello", MediaType: "", Privacy: "public", MediaName: "x.jpg",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for media type, got %v", err)
	}

	// missing privacy
	_, err = h.CreatePost(context.Background(), &feedpb.CreatePostRequest{
		AuthorId: 1, Text: "hello", MediaType: "image", Privacy: "", MediaName: "x.jpg",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for privacy, got %v", err)
	}

	// media bytes provided but missing media name
	_, err = h.CreatePost(context.Background(), &feedpb.CreatePostRequest{
		AuthorId: 1, Text: "", MediaData: []byte("x"), MediaType: "image", Privacy: "public", MediaName: "",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for media name, got %v", err)
	}
}

func TestCreatePost_Success(t *testing.T) {
	f := &fakeFeedSvc{
		CreatePostFn: func(ctx context.Context, authorID int64, text string, fileData []byte, fileName, mediaType, privacy string) (int64, error) {
			if authorID != 42 || text != "hello" || privacy != "public" {
				return 0, errors.New("bad args")
			}
			return 123, nil
		},
	}
	h := newHandlerWithFake(f)

	resp, err := h.CreatePost(context.Background(), &feedpb.CreatePostRequest{
		AuthorId: 42, Text: "hello", MediaType: "image", Privacy: "public", MediaName: "x.jpg",
	})
	if err != nil {
		t.Fatalf("CreatePost returned err: %v", err)
	}
	if resp.ContentId != 123 {
		t.Fatalf("expected contentId=123, got %d", resp.ContentId)
	}
}

func TestReactToContent_Success(t *testing.T) {
	called := false
	f := &fakeFeedSvc{
		ReactToContentFn: func(ctx context.Context, userID, contentID int64, reactionType string) error {
			if userID == 7 && contentID == 9 && reactionType == "like" {
				called = true
				return nil
			}
			return errors.New("unexpected args")
		},
	}
	h := newHandlerWithFake(f)

	resp, err := h.ReactToContent(context.Background(), &feedpb.ReactionRequest{
		UserId: 7, ContentId: 9, Type: "like",
	})
	if err != nil {
		t.Fatalf("ReactToContent returned err: %v", err)
	}
	if !called {
		t.Fatalf("expected service to be called")
	}
	if resp.GetMessage() == "" {
		t.Fatalf("expected success message")
	}
}

func TestGetTimeline_Success(t *testing.T) {
	f := &fakeFeedSvc{
		GetTimelineFn: func(ctx context.Context, userID int64) ([]dbmysql.Content, []string, error) {
			txt := "post text"
			return []dbmysql.Content{
					{ContentID: 1, AuthorID: 10, Type: "POST", TextContent: &txt, Privacy: "public"},
				},
				[]string{"http://localhost:8080/media/abc"},
				nil
		},
	}
	h := newHandlerWithFake(f)

	resp, err := h.GetTimeline(context.Background(), &feedpb.UserID{UserId: 10})
	if err != nil {
		t.Fatalf("GetTimeline returned err: %v", err)
	}
	if len(resp.Contents) != 1 || resp.Contents[0].ContentId != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestGetContent_InvalidID(t *testing.T) {
	h := newHandlerWithFake(&fakeFeedSvc{})
	_, err := h.GetContent(context.Background(), &feedpb.ContentID{ContentId: 0})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for bad content id, got %v", err)
	}
}
