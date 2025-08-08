package feed

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"GoSocial/internal/dbmysql"
)

// ---- In-memory fakes for repositories ----

type fakeContentRepo struct {
	store map[int64]dbmysql.Content
	next  int64

	CreateCalls int
	DeleteCalls int
}

func newFakeContentRepo() *fakeContentRepo {
	return &fakeContentRepo{store: map[int64]dbmysql.Content{}, next: 1}
}

func (r *fakeContentRepo) CreateContent(ctx context.Context, c *dbmysql.Content) error {
	r.CreateCalls++
	c.ContentID = r.next
	r.next++
	r.store[c.ContentID] = *c
	return nil
}

func (r *fakeContentRepo) GetContentByID(ctx context.Context, id int64) (*dbmysql.Content, error) {
	c, ok := r.store[id]
	if !ok {
		return nil, errors.New("not found")
	}
	// copy to avoid aliasing
	cc := c
	return &cc, nil
}

func (r *fakeContentRepo) ListUserContent(ctx context.Context, userID int64) ([]dbmysql.Content, error) {
	var out []dbmysql.Content
	for _, c := range r.store {
		if c.AuthorID == userID {
			// copy
			x := c
			out = append(out, x)
		}
	}
	return out, nil
}

func (r *fakeContentRepo) DeleteContent(ctx context.Context, id int64) error {
	r.DeleteCalls++
	delete(r.store, id)
	return nil
}

func (r *fakeContentRepo) ListExpiredStories(ctx context.Context, now time.Time) ([]dbmysql.Content, error) {
	// not used in these tests
	return nil, nil
}

type fakeMediaRepo struct {
	meta    map[int64]dbmysql.MediaRef
	data    map[int64][]byte
	next    int64
	deleted []int64

	CreateCalls  int
	DeleteCalls  int
	GetByIDCalls int
}

func newFakeMediaRepo() *fakeMediaRepo {
	return &fakeMediaRepo{
		meta: map[int64]dbmysql.MediaRef{},
		data: map[int64][]byte{},
		next: 1,
	}
}

func (m *fakeMediaRepo) CreateMediaRef(ctx context.Context, media *dbmysql.MediaRef, fileData []byte) error {
	m.CreateCalls++
	media.MediaRefID = m.next
	m.next++
	media.FilePath = "deadbeef" // pretend this is GridFS object id hex
	m.meta[media.MediaRefID] = *media
	m.data[media.MediaRefID] = append([]byte{}, fileData...) // copy
	return nil
}

func (m *fakeMediaRepo) GetMediaRefByID(ctx context.Context, id int64) (*dbmysql.MediaRef, []byte, error) {
	m.GetByIDCalls++
	meta, ok := m.meta[id]
	if !ok {
		return nil, nil, errors.New("not found")
	}
	data := append([]byte{}, m.data[id]...)
	cp := meta
	return &cp, data, nil
}

func (m *fakeMediaRepo) DeleteMedia(ctx context.Context, id int64) error {
	m.DeleteCalls++
	delete(m.meta, id)
	delete(m.data, id)
	m.deleted = append(m.deleted, id)
	return nil
}

type fakeReactionRepo struct {
	items map[string]dbmysql.Reaction
}

func newFakeReactionRepo() *fakeReactionRepo {
	return &fakeReactionRepo{items: map[string]dbmysql.Reaction{}}
}

func key(u, c int64) string { return string(rune(u)) + "|" + string(rune(c)) }

func (r *fakeReactionRepo) AddReaction(ctx context.Context, rx *dbmysql.Reaction) error {
	r.items[key(rx.UserID, rx.ContentID)] = *rx
	return nil
}
func (r *fakeReactionRepo) GetReactionsForContent(ctx context.Context, contentID int64) ([]dbmysql.Reaction, error) {
	var out []dbmysql.Reaction
	for _, v := range r.items {
		if v.ContentID == contentID {
			out = append(out, v)
		}
	}
	return out, nil
}
func (r *fakeReactionRepo) DeleteReaction(ctx context.Context, userID, contentID int64) error {
	delete(r.items, key(userID, contentID))
	return nil
}

// ---- Tests ----

func TestFeedService_CreatePost_WithMedia(t *testing.T) {
	cRepo := newFakeContentRepo()
	mRepo := newFakeMediaRepo()
	rRepo := newFakeReactionRepo()

	svc := &FeedService{
		contentRepo:  cRepo,
		mediaRepo:    mRepo,
		reactionRepo: rRepo,
		// UserClient not needed for this test
	}

	id, err := svc.CreatePost(context.Background(), 99, "hello", []byte("img-bytes"), "a.png", "image", "public")
	if err != nil {
		t.Fatalf("CreatePost err: %v", err)
	}
	if id == 0 {
		t.Fatalf("expected non-zero content id")
	}
	created, _ := cRepo.GetContentByID(context.Background(), id)
	if created.MediaRefID == nil {
		t.Fatalf("expected media_ref_id to be set")
	}
	if mRepo.CreateCalls != 1 || cRepo.CreateCalls != 1 {
		t.Fatalf("expected media and content to be created once")
	}
}

func TestFeedService_GetContent_ReturnsMediaURL(t *testing.T) {
	cRepo := newFakeContentRepo()
	mRepo := newFakeMediaRepo()
	rRepo := newFakeReactionRepo()
	svc := &FeedService{contentRepo: cRepo, mediaRepo: mRepo, reactionRepo: rRepo}

	// seed: create content with media
	txt := "t"
	content := &dbmysql.Content{AuthorID: 1, Type: "POST", TextContent: &txt, Privacy: "public"}
	_, _ = svc.CreateContent(context.Background(), content, []byte("data"), "image", "pic.png")

	got, url, err := svc.GetContent(context.Background(), content.ContentID)
	if err != nil {
		t.Fatalf("GetContent err: %v", err)
	}
	if got.ContentID != content.ContentID {
		t.Fatalf("mismatched content id")
	}
	if url == "" {
		t.Fatalf("expected non-empty media URL")
	}
	// sanity: url contains FilePath hex (fake "deadbeef")
	if url != GetMediaURL("deadbeef") {
		t.Fatalf("unexpected url: %s", url)
	}
}

func TestFeedService_ReactToContent_IdempotentSet(t *testing.T) {
	cRepo := newFakeContentRepo()
	mRepo := newFakeMediaRepo()
	rRepo := newFakeReactionRepo()
	svc := &FeedService{contentRepo: cRepo, mediaRepo: mRepo, reactionRepo: rRepo}

	// add twice with different types
	if err := svc.ReactToContent(context.Background(), 7, 101, "like"); err != nil {
		t.Fatalf("first react err: %v", err)
	}
	if err := svc.ReactToContent(context.Background(), 7, 101, "love"); err != nil {
		t.Fatalf("second react err: %v", err)
	}

	got, _ := rRepo.GetReactionsForContent(context.Background(), 101)
	if len(got) != 1 || got[0].Type != "love" {
		t.Fatalf("expected a single latest reaction 'love', got: %+v", got)
	}
}

func TestFeedService_GetUserContent_SelfView_ReturnsAllWithURLs(t *testing.T) {
	cRepo := newFakeContentRepo()
	mRepo := newFakeMediaRepo()
	rRepo := newFakeReactionRepo()
	svc := &FeedService{contentRepo: cRepo, mediaRepo: mRepo, reactionRepo: rRepo}

	// seed two contents for author=5: one with media, one without
	txt1 := "A"
	txt2 := "B"
	_, _ = svc.CreateContent(context.Background(), &dbmysql.Content{
		AuthorID: 5, Type: "POST", TextContent: &txt1, Privacy: "public",
	}, []byte("bytes"), "image", "pic.png")
	_, _ = svc.CreateContent(context.Background(), &dbmysql.Content{
		AuthorID: 5, Type: "POST", TextContent: &txt2, Privacy: "friends",
	}, nil, "", "")

	contents, urls, err := svc.GetUserContent(context.Background(), 5, 5) // self-view path (no UserClient)
	if err != nil {
		t.Fatalf("GetUserContent err: %v", err)
	}
	if len(contents) != 2 || len(urls) != 2 {
		t.Fatalf("expected 2 contents and 2 urls, got %d/%d", len(contents), len(urls))
	}
	// one with url, one empty
	hasURL := false
	hasEmpty := false
	for _, u := range urls {
		if u == "" {
			hasEmpty = true
		} else {
			hasURL = true
		}
	}
	if !(hasURL && hasEmpty) {
		t.Fatalf("expected mixed urls (one with media, one empty), got %v", urls)
	}
}

func TestFeedService_DeleteContent_RemovesMediaIfPresent(t *testing.T) {
	cRepo := newFakeContentRepo()
	mRepo := newFakeMediaRepo()
	rRepo := newFakeReactionRepo()
	svc := &FeedService{contentRepo: cRepo, mediaRepo: mRepo, reactionRepo: rRepo}

	// create with media
	txt := "X"
	_, _ = svc.CreateContent(context.Background(), &dbmysql.Content{
		AuthorID: 1, Type: "POST", TextContent: &txt, Privacy: "public",
	}, []byte("blob"), "image", "p.png")

	// find the created id
	var createdID int64
	for id := range cRepo.store {
		createdID = id
	}
	if createdID == 0 {
		t.Fatalf("seed failed")
	}

	if err := svc.DeleteContent(context.Background(), createdID); err != nil {
		t.Fatalf("DeleteContent err: %v", err)
	}
	if _, ok := cRepo.store[createdID]; ok {
		t.Fatalf("content still present after delete")
	}
	if mRepo.DeleteCalls != 1 {
		t.Fatalf("expected media delete to be called once, got %d", mRepo.DeleteCalls)
	}
}

// func TestFeedService_CreateStory_SetsDurationAndExpiration(t *testing.T) {
// 	cRepo := newFakeContentRepo()
// 	mRepo := newFakeMediaRepo()
// 	rRepo := newFakeReactionRepo()
// 	svc := &FeedService{contentRepo: cRepo, mediaRepo: mRepo, reactionRepo: rRepo}

// 	id, err := svc.CreateStory(context.Background(), 11, []byte("vid"), "video", "s.mp4", 60, "friends")
// 	if err != nil {
// 		t.Fatalf("CreateStory err: %v", err)
// 	}
// 	got, _ := cRepo.GetContentByID(context.Background(), id)
// 	if got.Type != "STORY" || got.Duration == nil || *got.Duration != 60 || got.Expiration == nil {
// 		t.Fatalf("story fields not set properly: %+v", got)
// 	}
// 	// expiration should be CreatedAt + duration (roughly). CreatedAt is set inside CreateContent,
// 	// but we can at least assert Expiration after CreatedAt.
// 	if got.Expiration.Before(got.CreatedAt) {
// 		t.Fatalf("expiration before createdAt")
// 	}
// }

// Guard: ensure repo fakes actually satisfy interfaces at compile time
var (
	_ Content   = (*fakeContentRepo)(nil)
	_ MediaRef  = (*fakeMediaRepo)(nil)
	_ Reactions = (*fakeReactionRepo)(nil)
)

// tiny helper to stop "unused" warnings in some editors
func assertEqual(t *testing.T, a, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("not equal: %v vs %v", a, b)
	}
}
