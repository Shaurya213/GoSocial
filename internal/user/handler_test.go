package user

import (
	"context"
	"errors"
	"testing"

	"gosocial/internal/dbmysql"
	pb "gosocial/api/v1/user"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Helper: context with user_id
func ctxWithUserID(uid uint64) context.Context {
	return context.WithValue(context.Background(), "user_id", uid)
}

// ---- Register ----
func TestHandler_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSvc := NewMockUserService(ctrl)
	h := NewHandler(mockSvc)
	ctx := context.Background()

	tests := []struct {
		name    string
		req     *pb.RegisterRequest
		setup   func()
		wantErr bool
		errCode codes.Code
	}{
		{
			name: "happy path",
			req: &pb.RegisterRequest{Handle: "alice", Email: "a@x.com", Password: "pwgood"},
			setup: func() {
				mockSvc.EXPECT().RegisterUser(ctx, "alice", "a@x.com", "pwgood").
					Return(&dbmysql.User{UserID: 2, Handle: "alice"}, "tok", nil)
			},
		},
		{
			name: "validation error",
			req: &pb.RegisterRequest{Handle: "!", Email: "bad", Password: ""},
			setup: func() {
				mockSvc.EXPECT().RegisterUser(ctx, "!", "bad", "").
					Return(nil, "", errors.New("invalid handle"))
			},
			wantErr: true, errCode: codes.InvalidArgument,
		},
		{
			name: "internal error",
			req: &pb.RegisterRequest{Handle: "bob", Email: "x@x.com", Password: "pwgood"},
			setup: func() {
				mockSvc.EXPECT().RegisterUser(ctx, "bob", "x@x.com", "pwgood").
					Return(nil, "", errors.New("db connection lost"))
			},
			wantErr: true, errCode: codes.InvalidArgument,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			resp, err := h.Register(ctx, tc.req)
			if tc.wantErr {
				require.Error(t, err)
				st, _ := status.FromError(err)
				require.Equal(t, tc.errCode, st.Code())
				require.NotNil(t, resp)
    			// require.False(t, resp.Success)

			} else {
				require.NoError(t, err)
				require.Equal(t, int64(2), resp.UserId)
			}
		})
	}
}

// ---- Login ----
func TestHandler_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSvc := NewMockUserService(ctrl)
	h := NewHandler(mockSvc)
	ctx := context.Background()

	tests := []struct {
		name    string
		req     *pb.LoginRequest
		setup   func()
		wantErr bool
		errCode codes.Code
	}{
		{
			name: "happy path",
			req: &pb.LoginRequest{Handle: "alice", Password: "pwgood"},
			setup: func() {
				mockSvc.EXPECT().LoginUser(ctx, "alice", "pwgood").
					Return(&dbmysql.User{UserID: 3, Handle: "alice"}, "tok", nil)
			},
		},
		{
			name: "wrong password",
			req: &pb.LoginRequest{Handle: "alice", Password: "badpw"},
			setup: func() {
				mockSvc.EXPECT().LoginUser(ctx, "alice", "badpw").
					Return(nil, "", errors.New("invalid handle or Password"))
			},
			wantErr: true, errCode: codes.Unauthenticated,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			resp, err := h.Login(ctx, tc.req)
			if tc.wantErr {
				require.Error(t, err)
				st, _ := status.FromError(err)
				require.Equal(t, tc.errCode, st.Code())
				require.NotNil(t, resp)
    			// require.False(t, resp.Success)

			} else {
				require.NoError(t, err)
				require.Equal(t, int64(3), resp.UserId)
			}
		})
	}
}

// ---- GetProfile ----
func TestHandler_GetProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSvc := NewMockUserService(ctrl)
	h := NewHandler(mockSvc)
	ctx := ctxWithUserID(10)

	tests := []struct {
		name    string
		ctx     context.Context
		setup   func()
		wantErr bool
		errCode codes.Code
	}{
		{
			name: "happy path",
			ctx:  ctx,
			setup: func() {
				mockSvc.EXPECT().GetProfile(ctx, uint64(10)).
					Return(&dbmysql.User{UserID: 10, Handle: "alice", Email: "b@b.com"}, nil)
			},
		},
		{
			name: "not found",
			ctx:  ctx,
			setup: func() {
				mockSvc.EXPECT().GetProfile(ctx, uint64(10)).
					Return(nil, errors.New("not found"))
			},
			wantErr: true, errCode: codes.NotFound,
		},
		{
			name:    "unauthenticated (no user_id)",
			ctx:     context.Background(),
			setup:   func() {},
			wantErr: true, errCode: codes.Unauthenticated,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			req := &pb.GetProfileRequest{}
			resp, err := h.GetProfile(tc.ctx, req)
			if tc.wantErr {
				require.Error(t, err)
				st, _ := status.FromError(err)
				require.Equal(t, tc.errCode, st.Code())
				require.NotNil(t, resp)
    			// require.False(t, resp.Success)
			} else {
				require.NoError(t, err)
				require.Equal(t, int64(10), resp.UserId)
			}
		})
	}
}

// ---- UpdateProfile ----
func TestHandler_UpdateProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSvc := NewMockUserService(ctrl)
	h := NewHandler(mockSvc)
	ctx := ctxWithUserID(20)

	tests := []struct {
		name      string
		ctx       context.Context
		req       *pb.UpdateProfileRequest
		setup     func()
		wantErr   bool
		errCode   codes.Code
		success   bool
	}{
		{
			name: "ok",
			ctx:  ctx,
			req:  &pb.UpdateProfileRequest{Email: "a@b.com"},
			setup: func() {
				mockSvc.EXPECT().UpdateProfile(ctx, uint64(20), "a@b.com", "", "").
					Return(nil)
			},
			success: true,
		},
		{
			name: "invalid email",
			ctx:  ctx,
			req:  &pb.UpdateProfileRequest{Email: "abc"},
			setup: func() {
				mockSvc.EXPECT().UpdateProfile(ctx, uint64(20), "abc", "", "").Return(errors.New("invalid email"))
			},
			wantErr: true, errCode: codes.InvalidArgument,
		},
		{
			name:    "unauthenticated",
			ctx:     context.Background(),
			req:     &pb.UpdateProfileRequest{Email: "a@b.com"},
			setup:   func() {},
			wantErr: true, errCode: codes.Unauthenticated,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			resp, err := h.UpdateProfile(tc.ctx, tc.req)
			if tc.wantErr {
				require.Error(t, err)
				st, _ := status.FromError(err)
				require.Equal(t, tc.errCode, st.Code())
				require.NotNil(t, resp)
    			require.False(t, resp.Success)

			} else {
				require.NoError(t, err)
				require.True(t, resp.Success)
			}
		})
	}
}

// ---- SendFriendRequest ----
func TestHandler_SendFriendRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSvc := NewMockUserService(ctrl)
	h := NewHandler(mockSvc)
	ctx := ctxWithUserID(100)

	tests := []struct {
		name    string
		ctx     context.Context
		target  int64
		setup   func()
		wantErr bool
		errCode codes.Code
	}{
		{
			name:   "ok",
			ctx:    ctx,
			target: 110,
			setup: func() {
				mockSvc.EXPECT().
					SendFriendRequest(ctx, uint64(100), uint64(110)).Return(nil)
			},
		},
		{
			name:   "already friends",
			ctx:    ctx,
			target: 111,
			setup: func() {
				mockSvc.EXPECT().
					SendFriendRequest(ctx, uint64(100), uint64(111)).
					Return(errors.New("already friends"))
			},
			wantErr: true, errCode: codes.FailedPrecondition,
		},
		{
			name:    "unauthenticated",
			ctx:     context.Background(),
			target:  110,
			setup:   func() {},
			wantErr: true, errCode: codes.Unauthenticated,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			resp, err := h.SendFriendRequest(tc.ctx, &pb.FriendRequest{TargetUserId: tc.target})
			if tc.wantErr {
				require.Error(t, err)
				st, _ := status.FromError(err)
				require.Equal(t, tc.errCode, st.Code())
				require.NotNil(t, resp)
    			require.False(t, resp.Success)
			} else {
				require.NoError(t, err)
				require.True(t, resp.Success)
			}
		})
	}
}

// ---- AcceptFriendRequest ----
func TestHandler_AcceptFriendRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSvc := NewMockUserService(ctrl)
	h := NewHandler(mockSvc)
	ctx := ctxWithUserID(122)

	tests := []struct {
		name     string
		ctx      context.Context
		requester int64
		setup    func()
		wantErr  bool
		errCode  codes.Code
	}{
		{
			name:      "ok",
			ctx:       ctx,
			requester: 123,
			setup: func() {
				mockSvc.EXPECT().
					AcceptFriendRequest(ctx, uint64(122), uint64(123)).Return(nil)
			},
		},
		{
			name:      "not found",
			ctx:       ctx,
			requester: 42,
			setup: func() {
				mockSvc.EXPECT().
					AcceptFriendRequest(ctx, uint64(122), uint64(42)).
					Return(errors.New("not found"))
			},
			wantErr: true, errCode: codes.FailedPrecondition,
		},
		{
			name:      "unauthenticated",
			ctx:       context.Background(),
			requester: 45,
			setup:     func() {},
			wantErr:   true,
			errCode:   codes.Unauthenticated,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			resp, err := h.AcceptFriendRequest(tc.ctx, &pb.FriendAcceptRequest{RequesterId: tc.requester})
			if tc.wantErr {
				require.Error(t, err)
				st, _ := status.FromError(err)
				require.Equal(t, tc.errCode, st.Code())
				require.NotNil(t, resp)
                require.False(t, resp.Success)


			} else {
				require.NoError(t, err)
				require.True(t, resp.Success)
			}
		})
	}
}

// ---- ListFriends ----
func TestHandler_ListFriends(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSvc := NewMockUserService(ctrl)
	h := NewHandler(mockSvc)
	ctx := ctxWithUserID(51)
	friends := []*dbmysql.User{{UserID: 9, Handle: "bob", Status: "active"}}

	tests := []struct {
		name    string
		ctx     context.Context
		setup   func()
		wantErr bool
		listLen int
	}{
		{
			name: "success (one friend)",
			ctx:  ctx,
			setup: func() {
				mockSvc.EXPECT().ListFriends(ctx, uint64(51)).
					Return(friends, nil)
			},
			listLen: 1,
		},
		{
			name: "repo error",
			ctx:  ctx,
			setup: func() {
				mockSvc.EXPECT().ListFriends(ctx, uint64(51)).
					Return(nil, errors.New("fail"))
			},
			wantErr: true,
		},
		{
			name: "no friends",
			ctx:  ctx,
			setup: func() {
				mockSvc.EXPECT().ListFriends(ctx, uint64(51)).
					Return([]*dbmysql.User{}, nil)
			},
			listLen: 0,
		},
		{
			name:    "unauthenticated",
			ctx:     context.Background(),
			setup:   func() {},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			resp, err := h.ListFriends(tc.ctx, &pb.UserID{})
			if tc.wantErr {
				require.Error(t, err)
				require.NotNil(t, resp)
                // require.False(t, resp.Success)

			} else {
				require.NoError(t, err)
				require.Len(t, resp.Friends, tc.listLen)
			}
		})
	}
}

// ---- RegisterDevice ----
func TestHandler_RegisterDevice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSvc := NewMockUserService(ctrl)
	h := NewHandler(mockSvc)
	ctx := ctxWithUserID(61)

	tests := []struct {
		name     string
		ctx      context.Context
		token    string
		platform string
		setup    func()
		wantErr  bool
		errCode  codes.Code
	}{
		{
			name:     "happy path",
			ctx:      ctx,
			token:    "fcmtok1", platform: "android",
			setup: func() {
				mockSvc.EXPECT().RegisterDevice(ctx, uint64(61), "fcmtok1", "android").Return(nil)
			},
		},
		{
			name:     "bad platform",
			ctx:      ctx,
			token:    "tok", platform: "windows",
			setup: func() {
				mockSvc.EXPECT().RegisterDevice(ctx, uint64(61), "tok", "windows").
					Return(errors.New("invalid platform"))
			},
			wantErr: true, errCode: codes.InvalidArgument,
		},
		{
			name:     "missing token",
			ctx:      ctx,
			token:    "", platform: "android",
			setup: func() {
				mockSvc.EXPECT().RegisterDevice(ctx, uint64(61), "", "android").
					Return(errors.New("device token required"))
			},
			wantErr: true, errCode: codes.InvalidArgument,
		},
		{
			name:    "unauthenticated",
			ctx:     context.Background(),
			token:   "tok", platform: "android",
			setup:   func() {},
			wantErr: true, errCode: codes.Unauthenticated,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			resp, err := h.RegisterDevice(tc.ctx, &pb.DeviceTokenRequest{DeviceToken: tc.token, Platform: tc.platform})
			if tc.wantErr {
				require.Error(t, err)
				st, _ := status.FromError(err)
				require.Equal(t, tc.errCode, st.Code())
				require.NotNil(t, resp)
    			require.False(t, resp.Success)
			} else {
				require.NoError(t, err)
				require.True(t, resp.Success)
			}
		})
	}
}

// ---- RemoveDevice ----
func TestHandler_RemoveDevice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSvc := NewMockUserService(ctrl)
	h := NewHandler(mockSvc)
	ctx := ctxWithUserID(101)

	tests := []struct {
		name    string
		ctx     context.Context
		token   string
		setup   func()
		wantErr bool
		errCode codes.Code
	}{
		{
			name:  "ok",
			ctx:   ctx,
			token: "tok10",
			setup: func() {
				mockSvc.EXPECT().RemoveDevice(ctx, "tok10").Return(nil)
			},
		},
		{
			name:  "fail",
			ctx:   ctx,
			token: "failtok",
			setup: func() {
				mockSvc.EXPECT().RemoveDevice(ctx, "failtok").
					Return(errors.New("not found"))
			},
			wantErr: true, errCode: codes.InvalidArgument,
		},
		{
			name:    "unauthenticated",
			ctx:     context.Background(),
			token:   "tok10",
			setup:   func() {},
			wantErr: true, errCode: codes.Unauthenticated,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			resp, err := h.RemoveDevice(tc.ctx, &pb.DeviceTokenRequest{DeviceToken: tc.token})
			if tc.wantErr {
				require.Error(t, err)
				st, _ := status.FromError(err)
				require.Equal(t, tc.errCode, st.Code())
			    require.NotNil(t, resp)
    			require.False(t, resp.Success)


			} else {
				require.NoError(t, err)
				require.True(t, resp.Success)
			}
		})
	}
}

// ---- GetUserDevices ----
func TestHandler_GetUserDevices(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSvc := NewMockUserService(ctrl)
	h := NewHandler(mockSvc)
	ctx := ctxWithUserID(77)
	devices := []*dbmysql.Device{{DeviceToken: "devtok1", Platform: "android"}}

	tests := []struct {
		name    string
		ctx     context.Context
		setup   func()
		wantErr bool
		listLen int
	}{
		{
			name: "success",
			ctx:  ctx,
			setup: func() {
				mockSvc.EXPECT().GetUserDevices(ctx, uint64(77)).Return(devices, nil)
			},
			listLen: 1,
		},
		{
			name: "empty list",
			ctx:  ctx,
			setup: func() {
				mockSvc.EXPECT().GetUserDevices(ctx, uint64(77)).Return([]*dbmysql.Device{}, nil)
			},
			listLen: 0,
		},
		{
			name: "repo error",
			ctx:  ctx,
			setup: func() {
				mockSvc.EXPECT().GetUserDevices(ctx, uint64(77)).Return(nil, errors.New("fail"))
			},
			wantErr: true,
		},
		{
			name:    "unauthenticated",
			ctx:     context.Background(),
			setup:   func() {},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			resp, err := h.GetUserDevices(tc.ctx, &pb.UserID{})
			if tc.wantErr {
				require.Error(t, err)
				require.NotNil(t, resp)
    			// require.False(t, resp.Success)


			} else {
				require.NoError(t, err)
				require.Len(t, resp.Devices, tc.listLen)
			}
		})
	}
}
