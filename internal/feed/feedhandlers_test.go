package feed

import (
	"context"
	"errors"
	"testing"
	"time"

	feedpb "gosocial/api/v1/feed"
	"gosocial/internal/dbmysql"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ---------- Fake service that satisfies FeedUsecase ----------

type fakeFeedSvc struct {
	CreatePostFn     func(ctx context.Context, authorID int64, text string, fileData []byte, fileName string, mediaType string, privacy string) (int64, error)
	CreateReelFn     func(ctx context.Context, authorID int64, caption string, fileData []byte, fileName string, durationSecs int, privacy string) (int64, error)
	CreateStoryFn    func(ctx context.Context, authorID int64, fileData []byte, mediaType string, mediaName string, durationSec int, privacy string) (int64, error)
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
func (f *fakeFeedSvc) GetReactions(ctx context.Context, cid int64) ([]dbmysql.Reaction, error) {
	return f.GetReactionsFn(ctx, cid)
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

func newHandlers(s *fakeFeedSvc) *FeedHandlers {
	return &FeedHandlers{FeedSvc: s}
}

func sptr(s string) *string { return &s }

// ---------- Tests ----------

func TestHandlers_CreatePost_ValidationsAndSuccess(t *testing.T) {
	h := newHandlers(&fakeFeedSvc{})
	// invalid author
	_, err := h.CreatePost(context.Background(), &feedpb.CreatePostRequest{
		AuthorId: 0, Text: "x", MediaType: "image", Privacy: "public", MediaName: "a.png",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("want InvalidArgument, got %v", err)
	}
	// no text & no media
	_, err = h.CreatePost(context.Background(), &feedpb.CreatePostRequest{
		AuthorId: 1, Text: "", MediaType: "image", Privacy: "public", MediaName: "a.png",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("want InvalidArgument(no body), got %v", err)
	}
	// missing media type
	_, err = h.CreatePost(context.Background(), &feedpb.CreatePostRequest{
		AuthorId: 1, Text: "x", MediaType: "", Privacy: "public", MediaName: "a.png",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("want InvalidArgument(media type), got %v", err)
	}
	// missing privacy
	_, err = h.CreatePost(context.Background(), &feedpb.CreatePostRequest{
		AuthorId: 1, Text: "x", MediaType: "image", Privacy: "", MediaName: "a.png",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("want InvalidArgument(privacy), got %v", err)
	}
	// missing media name with data
	_, err = h.CreatePost(context.Background(), &feedpb.CreatePostRequest{
		AuthorId: 1, Text: "", MediaData: []byte("d"), MediaType: "image", Privacy: "public", MediaName: "",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("want InvalidArgument(media name when data present), got %v", err)
	}

	// success
	ok := &fakeFeedSvc{
		CreatePostFn: func(ctx context.Context, a int64, t string, d []byte, n, mt, p string) (int64, error) {
			return 101, nil
		},
	}
	h = newHandlers(ok)
	resp, err := h.CreatePost(context.Background(), &feedpb.CreatePostRequest{
		AuthorId: 1, Text: "hello", MediaType: "image", Privacy: "public", MediaName: "a.png",
	})
	if err != nil || resp.ContentId != 101 {
		t.Fatalf("CreatePost success mismatch: resp=%+v err=%v", resp, err)
	}
}

func TestHandlers_CreateReel_ValidationsAndSuccess(t *testing.T) {
	ok := &fakeFeedSvc{
		CreateReelFn: func(ctx context.Context, a int64, c string, d []byte, n string, dur int, p string) (int64, error) {
			return 202, nil
		},
	}
	h := newHandlers(ok)

	// validations
	cases := []feedpb.CreateReelRequest{
		{AuthorId: 0, Caption: "x", MediaName: "v.mp4", DurationSecs: 5, Privacy: "public"},
		{AuthorId: 1, Caption: "", MediaData: nil, MediaName: "v.mp4", DurationSecs: 5, Privacy: "public"},
		{AuthorId: 1, Caption: "x", MediaName: "", DurationSecs: 5, Privacy: "public"},
		{AuthorId: 1, Caption: "x", MediaName: "v.mp4", DurationSecs: 0, Privacy: "public"},
		{AuthorId: 1, Caption: "x", MediaName: "v.mp4", DurationSecs: 5, Privacy: ""},
		{AuthorId: 1, Caption: "", MediaData: []byte("d"), MediaName: "", DurationSecs: 5, Privacy: "public"},
	}
	for i := range cases {
		_, err := h.CreateReel(context.Background(), &cases[i])
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("case %d: want InvalidArgument, got %v", i, err)
		}
	}

	// success
	resp, err := h.CreateReel(context.Background(), &feedpb.CreateReelRequest{
		AuthorId: 1, Caption: "cap", MediaData: []byte("v"), MediaName: "v.mp4", DurationSecs: 6, Privacy: "public",
	})
	if err != nil || resp.ContentId != 202 {
		t.Fatalf("CreateReel mismatch: resp=%+v err=%v", resp, err)
	}
}

func TestHandlers_CreateStory_ValidationsAndSuccess(t *testing.T) {
	ok := &fakeFeedSvc{
		CreateStoryFn: func(ctx context.Context, a int64, d []byte, mt, mn string, dur int, p string) (int64, error) {
			return 303, nil
		},
	}
	h := newHandlers(ok)

	// validations
	cases := []feedpb.CreateStoryRequest{
		{AuthorId: 0, MediaType: "image", MediaName: "p.png", DurationSecs: 5, Privacy: "friends"},
		{AuthorId: 1, MediaType: "image", MediaName: "", DurationSecs: 5, Privacy: "friends"},
		{AuthorId: 1, MediaType: "image", MediaName: "p.png", DurationSecs: 0, Privacy: "friends"},
		{AuthorId: 1, MediaType: "", MediaName: "p.png", DurationSecs: 5, Privacy: "friends"},
	}
	for i := range cases {
		_, err := h.CreateStory(context.Background(), &cases[i])
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("case %d: want InvalidArgument, got %v", i, err)
		}
	}

	// success
	resp, err := h.CreateStory(context.Background(), &feedpb.CreateStoryRequest{
		AuthorId: 1, MediaData: []byte("x"), MediaType: "image", MediaName: "p.png", DurationSecs: 12, Privacy: "friends",
	})
	if err != nil || resp.ContentId != 303 {
		t.Fatalf("CreateStory mismatch: resp=%+v err=%v", resp, err)
	}
}

func TestHandlers_Reactions_Flow(t *testing.T) {
	ok := &fakeFeedSvc{
		ReactToContentFn: func(ctx context.Context, u, c int64, r string) error { return nil },
		GetReactionsFn: func(ctx context.Context, cid int64) ([]dbmysql.Reaction, error) {
			return []dbmysql.Reaction{{UserID: 1, ContentID: 99, Type: "like"}}, nil
		},
		DeleteReactionFn: func(ctx context.Context, u, c int64) error { return nil },
	}
	h := newHandlers(ok)

	// react – bad params
	_, err := h.ReactToContent(context.Background(), &feedpb.ReactionRequest{UserId: 0, ContentId: 1, Type: "like"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument user")
	}

	// react – ok
	if _, err := h.ReactToContent(context.Background(), &feedpb.ReactionRequest{UserId: 1, ContentId: 99, Type: "like"}); err != nil {
		t.Fatalf("ReactToContent err: %v", err)
	}

	// list reactions – invalid id
	_, err = h.GetReactions(context.Background(), &feedpb.ContentID{ContentId: 0})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for content id")
	}

	// list reactions – ok
	r, err := h.GetReactions(context.Background(), &feedpb.ContentID{ContentId: 99})
	if err != nil || len(r.Reactions) != 1 || r.Reactions[0].Type != "like" {
		t.Fatalf("GetReactions mismatch: %+v err=%v", r, err)
	}

	// delete reaction – invalid
	_, err = h.DeleteReaction(context.Background(), &feedpb.DeleteReactionRequest{UserId: 0, ContentId: 1})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument on delete")
	}
	// delete reaction – ok
	if _, err := h.DeleteReaction(context.Background(), &feedpb.DeleteReactionRequest{UserId: 1, ContentId: 99}); err != nil {
		t.Fatalf("DeleteReaction err: %v", err)
	}
}

func TestHandlers_GetMediaRef_GetContent_DeleteContent_Timeline_UserContent(t *testing.T) {
	now := time.Now()
	// service with various behaviors
	ok := &fakeFeedSvc{
		GetMediaRefFn: func(ctx context.Context, id int64) (*dbmysql.MediaRef, error) {
			return &dbmysql.MediaRef{MediaRefID: uint(id), FileID: "deadbeef"}, nil
		},
		GetContentFn: func(ctx context.Context, id int64) (*dbmysql.Content, string, error) {
			txt := "hello"
			return &dbmysql.Content{ContentID: id, TextContent: &txt}, "url://x", nil
		},
		DeleteContentFn: func(ctx context.Context, id int64) error { return nil },
		GetTimelineFn: func(ctx context.Context, uid int64) ([]dbmysql.Content, []string, error) {
			// nil text to hit safeString(nil)
			return []dbmysql.Content{{ContentID: 1, AuthorID: 2, Type: "POST", TextContent: nil, Privacy: "public", CreatedAt: now}}, []string{""}, nil
		},
		GetUserContentFn: func(ctx context.Context, rid, tid int64) ([]dbmysql.Content, []string, error) {
			txt := "ok"
			return []dbmysql.Content{{ContentID: 9, AuthorID: tid, Type: "POST", TextContent: &txt, Privacy: "public", CreatedAt: now}}, []string{"m"}, nil
		},
	}

	h := newHandlers(ok)

	// GetMediaRef invalid
	if _, err := h.GetMediaRef(context.Background(), &feedpb.ContentID{ContentId: 0}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for GetMediaRef")
	}
	// GetMediaRef ok
	mr, err := h.GetMediaRef(context.Background(), &feedpb.ContentID{ContentId: 7})
	if err != nil || mr.FilePath != "deadbeef" {
		t.Fatalf("GetMediaRef mismatch: %+v err=%v", mr, err)
	}

	// GetContent invalid
	if _, err := h.GetContent(context.Background(), &feedpb.ContentID{ContentId: 0}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for GetContent")
	}
	// GetContent service error
	bad := newHandlers(&fakeFeedSvc{
		GetContentFn: func(ctx context.Context, id int64) (*dbmysql.Content, string, error) {
			return nil, "", errors.New("boom")
		},
	})
	if _, err := bad.GetContent(context.Background(), &feedpb.ContentID{ContentId: 5}); status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal on svc error")
	}
	// GetContent ok
	resp, err := h.GetContent(context.Background(), &feedpb.ContentID{ContentId: 3})
	if err != nil || resp.ContentId != 3 || resp.MediaUrl == "" || resp.Message != "hello" {
		t.Fatalf("GetContent mismatch: %+v err=%v", resp, err)
	}

	// DeleteContent invalid
	if _, err := h.DeleteContent(context.Background(), &feedpb.ContentID{ContentId: 0}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for DeleteContent")
	}
	// DeleteContent ok
	if _, err := h.DeleteContent(context.Background(), &feedpb.ContentID{ContentId: 3}); err != nil {
		t.Fatalf("DeleteContent err: %v", err)
	}

	// GetTimeline invalid
	if _, err := h.GetTimeline(context.Background(), &feedpb.UserID{UserId: 0}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for GetTimeline")
	}
	// GetTimeline ok (covers safeString(nil))
	tl, err := h.GetTimeline(context.Background(), &feedpb.UserID{UserId: 42})
	if err != nil || len(tl.Contents) != 1 || tl.Contents[0].Text != "" {
		t.Fatalf("timeline mismatch or safeString not applied: %+v err=%v", tl, err)
	}

	// GetUserContent invalid
	if _, err := h.GetUserContent(context.Background(), &feedpb.GetUserContentRequest{RequesterId: 0, TargetUserId: 1}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for GetUserContent")
	}
	// GetUserContent ok
	uc, err := h.GetUserContent(context.Background(), &feedpb.GetUserContentRequest{RequesterId: 2, TargetUserId: 2})
	if err != nil || len(uc.Contents) != 1 {
		t.Fatalf("GetUserContent mismatch: %+v err=%v", uc, err)
	}
}

// ---- ADDITIONAL TESTS FOR ERROR PATHS ----

func TestHandlers_ErrorBranches(t *testing.T) {
	cases := []struct {
		name string
		call func(h *FeedHandlers) error
	}{
		{
			name: "CreatePost service error",
			call: func(h *FeedHandlers) error {
				_, err := h.CreatePost(context.Background(),
					&feedpb.CreatePostRequest{AuthorId: 1, Text: "t", MediaType: "image", Privacy: "pub", MediaName: "n"})
				return err
			},
		},
		{
			name: "CreateReel service error",
			call: func(h *FeedHandlers) error {
				_, err := h.CreateReel(context.Background(),
					&feedpb.CreateReelRequest{AuthorId: 1, Caption: "c", MediaName: "n", DurationSecs: 1, Privacy: "pub"})
				return err
			},
		},
		{
			name: "CreateStory service error",
			call: func(h *FeedHandlers) error {
				_, err := h.CreateStory(context.Background(),
					&feedpb.CreateStoryRequest{AuthorId: 1, MediaName: "n", MediaType: "image", DurationSecs: 5, Privacy: "pub"})
				return err
			},
		},
		{
			name: "ReactToContent service error",
			call: func(h *FeedHandlers) error {
				_, err := h.ReactToContent(context.Background(),
					&feedpb.ReactionRequest{UserId: 1, ContentId: 2, Type: "like"})
				return err
			},
		},
		{
			name: "GetReactions service error",
			call: func(h *FeedHandlers) error {
				_, err := h.GetReactions(context.Background(), &feedpb.ContentID{ContentId: 1})
				return err
			},
		},
		{
			name: "DeleteReaction service error",
			call: func(h *FeedHandlers) error {
				_, err := h.DeleteReaction(context.Background(),
					&feedpb.DeleteReactionRequest{UserId: 1, ContentId: 2})
				return err
			},
		},
		{
			name: "GetMediaRef service error",
			call: func(h *FeedHandlers) error {
				_, err := h.GetMediaRef(context.Background(), &feedpb.ContentID{ContentId: 9})
				return err
			},
		},
		{
			name: "GetTimeline service error",
			call: func(h *FeedHandlers) error {
				_, err := h.GetTimeline(context.Background(), &feedpb.UserID{UserId: 5})
				return err
			},
		},
		{
			name: "GetUserContent service error",
			call: func(h *FeedHandlers) error {
				_, err := h.GetUserContent(context.Background(),
					&feedpb.GetUserContentRequest{RequesterId: 1, TargetUserId: 2})
				return err
			},
		},
	}

	ff := &fakeFeedSvc{
		CreatePostFn: func(context.Context, int64, string, []byte, string, string, string) (int64, error) {
			return 0, errors.New("fail")
		},
		CreateReelFn: func(context.Context, int64, string, []byte, string, int, string) (int64, error) {
			return 0, errors.New("fail")
		},
		CreateStoryFn: func(context.Context, int64, []byte, string, string, int, string) (int64, error) {
			return 0, errors.New("fail")
		},
		ReactToContentFn: func(context.Context, int64, int64, string) error { return errors.New("fail") },
		GetReactionsFn:   func(context.Context, int64) ([]dbmysql.Reaction, error) { return nil, errors.New("fail") },
		DeleteReactionFn: func(context.Context, int64, int64) error { return errors.New("fail") },
		GetMediaRefFn:    func(context.Context, int64) (*dbmysql.MediaRef, error) { return nil, errors.New("fail") },
		GetTimelineFn:    func(context.Context, int64) ([]dbmysql.Content, []string, error) { return nil, nil, errors.New("fail") },
		GetUserContentFn: func(context.Context, int64, int64) ([]dbmysql.Content, []string, error) {
			return nil, nil, errors.New("fail")
		},
	}
	h := newHandlers(ff)

	for _, c := range cases {
		if err := c.call(h); status.Code(err) != codes.Internal {
			t.Errorf("%s: expected Internal, got %v", c.name, err)
		}
	}
}
func TestHandlers_ServiceInternalErrors(t *testing.T) {
	ff := &fakeFeedSvc{
		GetContentFn: func(context.Context, int64) (*dbmysql.Content, string, error) {
			return nil, "", errors.New("fail-content")
		},
		GetMediaRefFn: func(context.Context, int64) (*dbmysql.MediaRef, error) {
			return nil, errors.New("fail-media")
		},
		GetTimelineFn: func(context.Context, int64) ([]dbmysql.Content, []string, error) {
			return nil, nil, errors.New("fail-timeline")
		},
		GetUserContentFn: func(context.Context, int64, int64) ([]dbmysql.Content, []string, error) {
			return nil, nil, errors.New("fail-usercontent")
		},
	}
	h := newHandlers(ff)

	if _, err := h.GetContent(context.Background(), &feedpb.ContentID{ContentId: 1}); status.Code(err) != codes.Internal {
		t.Errorf("GetContent: expected Internal, got %v", err)
	}
	if _, err := h.GetMediaRef(context.Background(), &feedpb.ContentID{ContentId: 2}); status.Code(err) != codes.Internal {
		t.Errorf("GetMediaRef: expected Internal, got %v", err)
	}
	if _, err := h.GetTimeline(context.Background(), &feedpb.UserID{UserId: 3}); status.Code(err) != codes.Internal {
		t.Errorf("GetTimeline: expected Internal, got %v", err)
	}
	if _, err := h.GetUserContent(context.Background(),
		&feedpb.GetUserContentRequest{RequesterId: 1, TargetUserId: 2}); status.Code(err) != codes.Internal {
		t.Errorf("GetUserContent: expected Internal, got %v", err)
	}
}

// --- Covers all handler happy paths that were previously 0.0% ---

func TestHandlers_AllHappyPaths(t *testing.T) {
	ff := &fakeFeedSvc{
		CreatePostFn: func(context.Context, int64, string, []byte, string, string, string) (int64, error) {
			return 101, nil
		},
		CreateReelFn: func(context.Context, int64, string, []byte, string, int, string) (int64, error) {
			return 202, nil
		},
		CreateStoryFn: func(context.Context, int64, []byte, string, string, int, string) (int64, error) {
			return 303, nil
		},
		ReactToContentFn: func(context.Context, int64, int64, string) error { return nil },
		GetReactionsFn: func(context.Context, int64) ([]dbmysql.Reaction, error) {
			return []dbmysql.Reaction{{UserID: 1, ContentID: 2, Type: "like"}}, nil
		},
		DeleteReactionFn: func(context.Context, int64, int64) error { return nil },
		GetMediaRefFn: func(context.Context, int64) (*dbmysql.MediaRef, error) {
			return &dbmysql.MediaRef{MediaRefID: 1, FileID: "file", Type: "image"}, nil
		},
		GetContentFn: func(context.Context, int64) (*dbmysql.Content, string, error) {
			txt := "sample"
			return &dbmysql.Content{ContentID: 5, TextContent: &txt}, "url://content", nil
		},
		DeleteContentFn: func(context.Context, int64) error { return nil },
		GetTimelineFn: func(context.Context, int64) ([]dbmysql.Content, []string, error) {
			return []dbmysql.Content{{ContentID: 7, AuthorID: 1, Privacy: "public", CreatedAt: time.Now()}}, []string{"url://tl"}, nil
		},
		GetUserContentFn: func(context.Context, int64, int64) ([]dbmysql.Content, []string, error) {
			return []dbmysql.Content{{ContentID: 9, AuthorID: 2, Privacy: "public", CreatedAt: time.Now()}}, []string{"url://uc"}, nil
		},
	}
	h := newHandlers(ff)

	cases := []struct {
		name string
		run  func() error
	}{
		{"CreatePost", func() error {
			_, e := h.CreatePost(context.Background(), &feedpb.CreatePostRequest{AuthorId: 1, Text: "t", MediaType: "image", Privacy: "public", MediaName: "m"})
			return e
		}},
		{"CreateReel", func() error {
			_, e := h.CreateReel(context.Background(), &feedpb.CreateReelRequest{AuthorId: 1, Caption: "c", MediaName: "m", DurationSecs: 5, Privacy: "public"})
			return e
		}},
		{"CreateStory", func() error {
			_, e := h.CreateStory(context.Background(), &feedpb.CreateStoryRequest{AuthorId: 1, MediaType: "image", MediaName: "m", DurationSecs: 5, Privacy: "public"})
			return e
		}},
		{"ReactToContent", func() error {
			_, e := h.ReactToContent(context.Background(), &feedpb.ReactionRequest{UserId: 1, ContentId: 2, Type: "like"})
			return e
		}},
		{"GetReactions", func() error { _, e := h.GetReactions(context.Background(), &feedpb.ContentID{ContentId: 2}); return e }},
		{"DeleteReaction", func() error {
			_, e := h.DeleteReaction(context.Background(), &feedpb.DeleteReactionRequest{UserId: 1, ContentId: 2})
			return e
		}},
		{"GetMediaRef", func() error { _, e := h.GetMediaRef(context.Background(), &feedpb.ContentID{ContentId: 1}); return e }},
		{"GetContent", func() error { _, e := h.GetContent(context.Background(), &feedpb.ContentID{ContentId: 5}); return e }},
		{"DeleteContent", func() error { _, e := h.DeleteContent(context.Background(), &feedpb.ContentID{ContentId: 5}); return e }},
		{"GetTimeline", func() error { _, e := h.GetTimeline(context.Background(), &feedpb.UserID{UserId: 1}); return e }},
		{"GetUserContent", func() error {
			_, e := h.GetUserContent(context.Background(), &feedpb.GetUserContentRequest{RequesterId: 1, TargetUserId: 2})
			return e
		}},
	}

	for _, c := range cases {
		if err := c.run(); err != nil {
			t.Errorf("%s failed: %v", c.name, err)
		}
	}
}
