package feed

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	userpb "GoSocial/api/v1/user"
	"GoSocial/internal/dbmysql"
	"google.golang.org/grpc"
)

// ---------- Fakes for Content/Media/Reaction repos ----------

type fakeContentRepo struct {
	m    map[int64]dbmysql.Content
	next int64
}

func newFakeContentRepo() *fakeContentRepo {
	return &fakeContentRepo{m: map[int64]dbmysql.Content{}, next: 1}
}
func (r *fakeContentRepo) CreateContent(ctx context.Context, c *dbmysql.Content) error {
	if c.ContentID == 0 {
		c.ContentID = r.next
		r.next++
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	cp := *c
	r.m[c.ContentID] = cp
	return nil
}
func (r *fakeContentRepo) GetContentByID(ctx context.Context, id int64) (*dbmysql.Content, error) {
	c, ok := r.m[id]
	if !ok {
		return nil, errors.New("not found")
	}
	cp := c
	return &cp, nil
}
func (r *fakeContentRepo) ListUserContent(ctx context.Context, userID int64) ([]dbmysql.Content, error) {
	var out []dbmysql.Content
	for _, v := range r.m {
		if v.AuthorID == userID {
			x := v
			out = append(out, x)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
func (r *fakeContentRepo) DeleteContent(ctx context.Context, id int64) error {
	delete(r.m, id)
	return nil
}
func (r *fakeContentRepo) ListExpiredStories(ctx context.Context, now time.Time) ([]dbmysql.Content, error) {
	var out []dbmysql.Content
	for _, v := range r.m {
		if v.Type == "STORY" && v.Expiration != nil && !v.Expiration.After(now) {
			out = append(out, v)
		}
	}
	return out, nil
}

type fakeMediaRepo struct {
	meta      map[int64]dbmysql.MediaRef
	data      map[int64][]byte
	next      int64
	deleteErr error
}

func newFakeMediaRepo() *fakeMediaRepo {
	return &fakeMediaRepo{meta: map[int64]dbmysql.MediaRef{}, data: map[int64][]byte{}, next: 1}
}
func (m *fakeMediaRepo) CreateMediaRef(ctx context.Context, media *dbmysql.MediaRef, fileData []byte) error {
	media.MediaRefID = m.next
	m.next++
	if media.FilePath == "" {
		media.FilePath = "deadbeef"
	}
	m.meta[media.MediaRefID] = *media
	m.data[media.MediaRefID] = append([]byte{}, fileData...)
	return nil
}
func (m *fakeMediaRepo) GetMediaRefByID(ctx context.Context, id int64) (*dbmysql.MediaRef, []byte, error) {
	meta, ok := m.meta[id]
	if !ok {
		return nil, nil, errors.New("not found")
	}
	cp := meta
	return &cp, append([]byte{}, m.data[id]...), nil
}
func (m *fakeMediaRepo) DeleteMedia(ctx context.Context, id int64) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.meta, id)
	delete(m.data, id)
	return nil
}

type fakeReactionRepo struct{ m map[string]dbmysql.Reaction }

func newFakeReactionRepo() *fakeReactionRepo {
	return &fakeReactionRepo{m: map[string]dbmysql.Reaction{}}
}
func key(u, c int64) string { return fmt.Sprintf("%d|%d", u, c) }

func (r *fakeReactionRepo) AddReaction(ctx context.Context, rx *dbmysql.Reaction) error {
	r.m[key(rx.UserID, rx.ContentID)] = *rx
	return nil
}
func (r *fakeReactionRepo) GetReactionsForContent(ctx context.Context, contentID int64) ([]dbmysql.Reaction, error) {
	var out []dbmysql.Reaction
	for _, v := range r.m {
		if v.ContentID == contentID {
			out = append(out, v)
		}
	}
	return out, nil
}
func (r *fakeReactionRepo) DeleteReaction(ctx context.Context, u, c int64) error {
	delete(r.m, key(u, c))
	return nil
}

// ---------- Fake user client (ListFriends only) ----------

type fakeUserClient struct {
	userpb.UserServiceClient
	ListFn func(ctx context.Context, in *userpb.UserID, opts ...grpc.CallOption) (*userpb.FriendList, error)
}

func (f *fakeUserClient) ListFriends(ctx context.Context, in *userpb.UserID, opts ...grpc.CallOption) (*userpb.FriendList, error) {
	return f.ListFn(ctx, in, opts...)
}

// ---------- Tests ----------

func TestService_CreateContent_NoMedia_And_WithMedia(t *testing.T) {
	cRepo := newFakeContentRepo()
	mRepo := newFakeMediaRepo()
	rRepo := newFakeReactionRepo()
	svc := &FeedService{contentRepo: cRepo, mediaRepo: mRepo, reactionRepo: rRepo}

	// no media
	txt := "hello"
	id1, err := svc.CreateContent(context.Background(), &dbmysql.Content{
		AuthorID: 1, Type: "POST", TextContent: &txt, Privacy: "public",
	}, nil, "", "")
	if err != nil || id1 == 0 {
		t.Fatalf("CreateContent(no media) err=%v id=%d", err, id1)
	}

	// with media
	txt2 := "hi"
	id2, err := svc.CreateContent(context.Background(), &dbmysql.Content{
		AuthorID: 2, Type: "POST", TextContent: &txt2, Privacy: "public",
	}, []byte("blob"), "image", "pic.png")
	if err != nil || id2 == 0 {
		t.Fatalf("CreateContent(with media) err=%v id=%d", err, id2)
	}
	got, _, _ := svc.GetContent(context.Background(), id2)
	if got.MediaRefID == nil {
		t.Fatalf("expected media_ref_id to be set")
	}
}

func TestService_DeleteContent_MediaDeleteErrorIsIgnored(t *testing.T) {
	cRepo := newFakeContentRepo()
	mRepo := newFakeMediaRepo()
	rRepo := newFakeReactionRepo()
	svc := &FeedService{contentRepo: cRepo, mediaRepo: mRepo, reactionRepo: rRepo}

	// create content with media
	txt := "x"
	id, err := svc.CreateContent(context.Background(), &dbmysql.Content{
		AuthorID: 9, Type: "POST", TextContent: &txt, Privacy: "public",
	}, []byte("media"), "image", "a.png")
	if err != nil {
		t.Fatal(err)
	}
	mRepo.deleteErr = errors.New("gridfs down")

	if err := svc.DeleteContent(context.Background(), id); err != nil {
		t.Fatalf("DeleteContent returned err: %v", err)
	}
	if _, ok := cRepo.m[id]; ok {
		t.Fatalf("content should be removed even if media deletion failed")
	}
}

func TestService_CreateReel_And_CreateStory(t *testing.T) {
	cRepo := newFakeContentRepo()
	mRepo := newFakeMediaRepo()
	rRepo := newFakeReactionRepo()
	svc := &FeedService{contentRepo: cRepo, mediaRepo: mRepo, reactionRepo: rRepo}

	// reel
	idr, err := svc.CreateReel(context.Background(), 1, "cap", []byte("v"), "r.mp4", 12, "public")
	if err != nil || idr == 0 {
		t.Fatalf("CreateReel err=%v id=%d", err, idr)
	}

	// story
	ids, err := svc.CreateStory(context.Background(), 2, []byte("p"), "image", "s.png", 60, "friends")
	if err != nil || ids == 0 {
		t.Fatalf("CreateStory err=%v id=%d", err, ids)
	}
	st, _ := cRepo.GetContentByID(context.Background(), ids)
	if st.Type != "STORY" || st.Duration == nil || *st.Duration != 60 || st.Expiration == nil {
		t.Fatalf("story fields wrong: %+v", st)
	}
}

func TestService_CreateMediaRef_And_GetMediaRef(t *testing.T) {
	cRepo := newFakeContentRepo()
	mRepo := newFakeMediaRepo()
	rRepo := newFakeReactionRepo()
	svc := &FeedService{contentRepo: cRepo, mediaRepo: mRepo, reactionRepo: rRepo}

	m := &dbmysql.MediaRef{Type: "image", FileName: "a.png", UploadedBy: 1}
	id, err := svc.CreateMediaRef(context.Background(), m, []byte("xx"))
	if err != nil || id == 0 {
		t.Fatalf("CreateMediaRef err=%v id=%d", err, id)
	}
	meta, data, err := svc.GetMediaRef(context.Background(), id)
	if err != nil || meta.FilePath != "deadbeef" || string(data) != "xx" {
		t.Fatalf("GetMediaRef mismatch: meta=%+v data=%s err=%v", meta, string(data), err)
	}
}

func TestService_Reactions_SetSemantics(t *testing.T) {
	cRepo := newFakeContentRepo()
	mRepo := newFakeMediaRepo()
	rRepo := newFakeReactionRepo()
	svc := &FeedService{contentRepo: cRepo, mediaRepo: mRepo, reactionRepo: rRepo}

	// like then change to love (idempotent "set")
	if err := svc.ReactToContent(context.Background(), 1, 10, "like"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ReactToContent(context.Background(), 1, 10, "love"); err != nil {
		t.Fatal(err)
	}
	rxs, _ := svc.GetReactions(context.Background(), 10)
	if len(rxs) != 1 || rxs[0].Type != "love" {
		t.Fatalf("want single love, got %+v", rxs)
	}

	// delete
	if err := svc.DeleteReaction(context.Background(), 1, 10); err != nil {
		t.Fatalf("DeleteReaction err: %v", err)
	}
}

func TestService_GetTimeline_SortsAndBuildsMediaURLs(t *testing.T) {
	cRepo := newFakeContentRepo()
	mRepo := newFakeMediaRepo()
	rRepo := newFakeReactionRepo()

	// seed content for self(1) and friend(2)
	t1 := "friend"
	t2 := "self"
	_ = cRepo.CreateContent(context.Background(), &dbmysql.Content{
		AuthorID: 2, Type: "POST", TextContent: &t1, Privacy: "public", CreatedAt: time.Now().Add(-2 * time.Minute),
	})
	// self newer with media
	_, _ = (&FeedService{contentRepo: cRepo, mediaRepo: mRepo, reactionRepo: rRepo}).CreateContent(context.Background(),
		&dbmysql.Content{AuthorID: 1, Type: "POST", TextContent: &t2, Privacy: "public", CreatedAt: time.Now().Add(-1 * time.Minute)},
		[]byte("m"), "image", "p.png",
	)

	uc := &fakeUserClient{
		ListFn: func(ctx context.Context, in *userpb.UserID, _ ...grpc.CallOption) (*userpb.FriendList, error) {
			return &userpb.FriendList{
				Friends: []*userpb.Friend{{UserId: 2}}, // <-- use Friend, not UserID
			}, nil
		},
	}

	svc := &FeedService{contentRepo: cRepo, mediaRepo: mRepo, reactionRepo: rRepo, UserClient: uc}

	cs, urls, err := svc.GetTimeline(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetTimeline err: %v", err)
	}
	// newest first -> self first
	if len(cs) != 2 || cs[0].AuthorID != 1 {
		t.Fatalf("unexpected order: %+v", cs)
	}
	if urls[0] == "" || urls[1] != "" {
		t.Fatalf("unexpected urls: %v", urls)
	}
}

func TestService_GetUserContent_FriendshipFilter(t *testing.T) {
	cRepo := newFakeContentRepo()
	mRepo := newFakeMediaRepo()
	rRepo := newFakeReactionRepo()
	// target user 7 has public + friends posts
	pub := "pub"
	fr := "fr"
	_ = cRepo.CreateContent(context.Background(), &dbmysql.Content{
		AuthorID: 7, Type: "POST", TextContent: &pub, Privacy: "public", CreatedAt: time.Now().Add(-2 * time.Minute),
	})
	_ = cRepo.CreateContent(context.Background(), &dbmysql.Content{
		AuthorID: 7, Type: "POST", TextContent: &fr, Privacy: "friends", CreatedAt: time.Now().Add(-1 * time.Minute),
	})

	uc := &fakeUserClient{
		ListFn: func(ctx context.Context, in *userpb.UserID, _ ...grpc.CallOption) (*userpb.FriendList, error) {
			return &userpb.FriendList{
				Friends: []*userpb.Friend{{UserId: 8}}, // <-- use Friend, not UserID
			}, nil
		},
	}

	svc := &FeedService{contentRepo: cRepo, mediaRepo: mRepo, reactionRepo: rRepo, UserClient: uc}

	// requester is friend -> sees both
	cs, _, err := svc.GetUserContent(context.Background(), 8, 7)
	if err != nil || len(cs) != 2 {
		t.Fatalf("friend should see 2 posts, got %d err=%v", len(cs), err)
	}
	// requester is not friend -> sees only public
	cs2, _, err := svc.GetUserContent(context.Background(), 9, 7)
	if err != nil || len(cs2) != 1 || cs2[0].Privacy != "public" {
		t.Fatalf("non-friend should see 1 public, got %d (%+v) err=%v", len(cs2), cs2, err)
	}
}

func TestService_GetContent_NotFound(t *testing.T) {
	cRepo := newFakeContentRepo()
	mRepo := newFakeMediaRepo()
	rRepo := newFakeReactionRepo()
	svc := &FeedService{contentRepo: cRepo, mediaRepo: mRepo, reactionRepo: rRepo}

	_, _, err := svc.GetContent(context.Background(), 42)
	if err == nil {
		t.Fatal("expected error for missing content")
	}
}

// ---- ADDITIONAL SERVICE TESTS ----
//
//func TestService_GetContent_MediaFetchError(t *testing.T) {
//	cRepo := newFakeContentRepo()
//	mRepo := newFakeMediaRepo()
//	rRepo := newFakeReactionRepo()
//
//	txt := "x"
//	id, _ := cRepo.CreateContent(context.Background(),
//		&dbmysql.Content{AuthorID: 1, Type: "POST", TextContent: &txt, Privacy: "public"})
//	badID := int64(99)
//	c := cRepo.m[id]
//	c.MediaRefID = &badID
//	cRepo.m[id] = c
//
//	svc := &FeedService{contentRepo: cRepo, mediaRepo: mRepo, reactionRepo: rRepo}
//	_, url, err := svc.GetContent(context.Background(), id)
//	if err == nil && url != "" {
//		t.Fatalf("expected empty url on media fetch fail")
//	}
//}

//func TestService_DeleteContent_NoMedia(t *testing.T) {
//	cRepo := newFakeContentRepo()
//	mRepo := newFakeMediaRepo()
//	rRepo := newFakeReactionRepo()
//	txt := "only text"
//	id, _ := cRepo.CreateContent(context.Background(),
//		&dbmysql.Content{AuthorID: 2, Type: "POST", TextContent: &txt, Privacy: "public"})
//	svc := &FeedService{contentRepo: cRepo, mediaRepo: mRepo, reactionRepo: rRepo}
//	if err := svc.DeleteContent(context.Background(), id); err != nil {
//		t.Fatalf("unexpected err: %v", err)
//	}
//}

func TestService_GetTimeline_ListFriendsError(t *testing.T) {
	cRepo := newFakeContentRepo()
	mRepo := newFakeMediaRepo()
	rRepo := newFakeReactionRepo()
	uc := &fakeUserClient{
		ListFn: func(ctx context.Context, in *userpb.UserID, _ ...grpc.CallOption) (*userpb.FriendList, error) {
			return nil, errors.New("fail")
		},
	}
	svc := &FeedService{contentRepo: cRepo, mediaRepo: mRepo, reactionRepo: rRepo, UserClient: uc}
	_, _, err := svc.GetTimeline(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestService_GetUserContent_ListFriendsError(t *testing.T) {
	cRepo := newFakeContentRepo()
	mRepo := newFakeMediaRepo()
	rRepo := newFakeReactionRepo()
	txt := "public"
	_ = cRepo.CreateContent(context.Background(),
		&dbmysql.Content{AuthorID: 5, Type: "POST", TextContent: &txt, Privacy: "friends"})
	uc := &fakeUserClient{
		ListFn: func(ctx context.Context, in *userpb.UserID, _ ...grpc.CallOption) (*userpb.FriendList, error) {
			return nil, errors.New("fail")
		},
	}
	svc := &FeedService{contentRepo: cRepo, mediaRepo: mRepo, reactionRepo: rRepo, UserClient: uc}
	_, _, err := svc.GetUserContent(context.Background(), 1, 5)
	if err == nil {
		t.Fatal("expected error from ListFriends")
	}
}

func TestService_ExpiredStoryCleaner(t *testing.T) {
	cRepo := newFakeContentRepo()
	mRepo := newFakeMediaRepo()
	rRepo := newFakeReactionRepo()
	expiration := time.Now().Add(-1 * time.Minute)
	_ = cRepo.CreateContent(context.Background(),
		&dbmysql.Content{AuthorID: 1, Type: "STORY", Privacy: "public", Expiration: &expiration})
	svc := &FeedService{contentRepo: cRepo, mediaRepo: mRepo, reactionRepo: rRepo}
	// run once manually
	expired, _ := svc.contentRepo.ListExpiredStories(context.Background(), time.Now())
	if len(expired) == 0 {
		t.Fatal("expected expired story present")
	}
	for _, story := range expired {
		_ = svc.DeleteContent(context.Background(), story.ContentID)
	}
}
