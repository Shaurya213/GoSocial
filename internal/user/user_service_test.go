package user

import (
	"context"
	"errors"
	"gosocial/internal/common"
	"gosocial/internal/dbmysql"
	"testing"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserService_RegisterUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()


	mockUserRepo := NewMockUserRepository(ctrl)
	mockFriendRepo := NewMockFriendRepository(ctrl)
	mockDeviceRepo := NewMockDeviceRepository(ctrl)
	svc := NewUserService(mockUserRepo, mockFriendRepo, mockDeviceRepo)
	ctx := context.Background()

	
	tests := []struct {
		name        string
		handle      string
		email       string
		password    string
		setup       func()
		wantErr     bool
		errContains string
	}{
		{
			name:     "success",
			handle:   "alice",
			email:    "alice@example.com",
			password: "Password123",
			setup: func() {
				mockUserRepo.EXPECT().CheckUserExists(ctx, "alice").Return(false, nil)
				mockUserRepo.EXPECT().CreateUser(ctx, gomock.Any()).DoAndReturn(
					func(_ context.Context, u *dbmysql.User) error {
						u.UserID = 1 // fake assignment
						return nil
					})
			},
		},
		{
			name:     "duplicate handle",
			handle:   "bob",
			email:    "bob@example.com",
			password: "Password123",
			setup: func() {
				mockUserRepo.EXPECT().CheckUserExists(ctx, "bob").Return(true, nil)
			},
			wantErr:     true,
			errContains: "exists",
		},
		{
			name:        "invalid handle",
			handle:      "!",
			email:       "x@y.com",
			password:    "Password123",
			setup:       func() {},
			wantErr:     true,
			errContains: "Handle",
		},

		{
			name: "invalid email",
			handle: "alicegood",
			email: "bademail",
			password: "Password123",
			setup: func(){},
			wantErr: true,
			errContains: "email",
		},
		{
			name: "invalid password",
			handle: "alicia",
			email: "alic@g.com",
			password: "short",
			setup: func(){},
			wantErr: true,
			errContains: "password",
		},
		{
			name: "repo failure exist check",
			handle: "alicefail",
			email: "alice@fail.com",
			password: "Password123",
			setup: func() {
			mockUserRepo.EXPECT().CheckUserExists(ctx, "alicefail").Return(false, errors.New("db is down"))
			},
			wantErr: true,
			errContains: "db is down",
		},
		{
			name: "repo failure create user",
			handle: "alicefail2",
			email: "alice2@fail.com",
			password: "Password123",
			setup: func() {
			mockUserRepo.EXPECT().CheckUserExists(ctx, "alicefail2").Return(false, nil)
			mockUserRepo.EXPECT().CreateUser(ctx, gomock.Any()).Return(errors.New("create fail"))
			},
			wantErr: true,
			errContains: "create fail",
		},

	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			user, token, err := svc.RegisterUser(ctx, tc.handle, tc.email, tc.password)
			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
				require.Nil(t, user)
				require.Empty(t, token)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				require.NotEmpty(t, token)
				require.Equal(t, tc.handle, user.Handle)
			}
		})
	}
}

// This test Verify authentication works for
// Correct credentials (“happy path”)
// Incorrect password
// Handle not found
// User not active
func TestUserService_LoginUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockUserRepo := NewMockUserRepository(ctrl)
	mockFriendRepo := NewMockFriendRepository(ctrl)
	mockDeviceRepo := NewMockDeviceRepository(ctrl)
	svc := NewUserService(mockUserRepo, mockFriendRepo, mockDeviceRepo)
	ctx := context.Background()

	// Setup example user and hash
	hash, _ := common.HashPassword("GoodPassword1")
	activeUser := &dbmysql.User{UserID: 2, Handle: "bob", PasswordHash: hash, Status: "active"}
	bannedUser := &dbmysql.User{UserID: 3, Handle: "banned", PasswordHash: hash, Status: "banned"}

	tests := []struct {
		name        string
		handle      string
		password    string
		setup       func()
		wantErr     bool
		errContains string
	}{
		{
			name:     "success",
			handle:   "bob",
			password: "GoodPassword1",
			setup: func() {
				mockUserRepo.EXPECT().GetUserByHandle(ctx, "bob").Return(activeUser, nil)
			},
		},
		{
			name:     "bad password",
			handle:   "bob",
			password: "WrongPassword",
			setup: func() {
				mockUserRepo.EXPECT().GetUserByHandle(ctx, "bob").Return(activeUser, nil)
			},
			wantErr:     true,
			errContains: "invalid password",
		},
		{
			name:     "user not found",
			handle:   "nobody",
			password: "anything",
			setup: func() {
				mockUserRepo.EXPECT().GetUserByHandle(ctx, "nobody").Return(nil, gorm.ErrRecordNotFound)
			},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:     "user banned",
			handle:   "banned",
			password: "GoodPassword1",
			setup: func() {
				mockUserRepo.EXPECT().GetUserByHandle(ctx, "banned").Return(bannedUser, nil)
			},
			wantErr:     true,
			errContains: "not active",
		},
		

	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			user, token, err := svc.LoginUser(ctx, tc.handle, tc.password)
			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
				require.Nil(t, user)
				require.Empty(t, token)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				require.NotEmpty(t, token)
				require.Equal(t, tc.handle, user.Handle)
			}
		})
	}
}


func TestUserService_GetProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockUserRepo := NewMockUserRepository(ctrl)
	mockFriendRepo := NewMockFriendRepository(ctrl)
	mockDeviceRepo := NewMockDeviceRepository(ctrl)
	svc := NewUserService(mockUserRepo, mockFriendRepo, mockDeviceRepo)
	ctx := context.Background()

	correctUser := &dbmysql.User{
		UserID: 1,
		Handle: "alice",
		Email:  "alice@mail.com",
		Status: "active",
	}

	tests := []struct {
		name        string
		userID      uint64
		setup       func()
		wantErr     bool
		errContains string
	}{
		{
			name:   "user found",
			userID: 1,
			setup: func() {
				mockUserRepo.EXPECT().GetUserByID(ctx, uint64(1)).Return(correctUser, nil)
			},
		},
		{
			name:   "user not found",
			userID: 42,
			setup: func() {
				mockUserRepo.EXPECT().GetUserByID(ctx, uint64(42)).Return(nil, gorm.ErrRecordNotFound)
			},
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			user, err := svc.GetProfile(ctx, tc.userID)
			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
				require.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				require.Equal(t, tc.userID, user.UserID)
			}
		})
	}
}

func TestUserService_UpdateProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockUserRepo := NewMockUserRepository(ctrl)
	mockFriendRepo := NewMockFriendRepository(ctrl)
	mockDeviceRepo := NewMockDeviceRepository(ctrl)
	svc := NewUserService(mockUserRepo, mockFriendRepo, mockDeviceRepo)
	ctx := context.Background()

	// Mock a user in the DB
	origUser := &dbmysql.User{
		UserID: 1, Handle: "alice", Email: "old@email.com", Phone: "111", ProfileDetails: "old", Status: "active",
	}

	tests := []struct {
		name        string
		userID      uint64
		newEmail    string
		newPhone    string
		newProfile  string
		setup       func()
		wantErr     bool
		errContains string
	}{
		{
			name:       "happy update all",
			userID:     1,
			newEmail:   "alice@go.com",
			newPhone:   "9999",
			newProfile: "bio updated",
			setup: func() {
				mockUserRepo.EXPECT().GetUserByID(ctx, uint64(1)).Return(origUser, nil)
				// Verifies that fields are updated
				mockUserRepo.EXPECT().UpdateUser(ctx, gomock.AssignableToTypeOf(&dbmysql.User{})).
					DoAndReturn(func(_ context.Context, u *dbmysql.User) error {
						require.Equal(t, "alice@go.com", u.Email)
						require.Equal(t, "9999", u.Phone)
						require.Equal(t, "bio updated", u.ProfileDetails)
						return nil
					})
			},
		},
		{
			name:     "update user not found",
			userID:   9,
			newEmail: "a@x.com",
			setup: func() {
				mockUserRepo.EXPECT().GetUserByID(ctx, uint64(9)).Return(nil, gorm.ErrRecordNotFound)
			},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:     "invalid email",
			userID:   1,
			newEmail: "bad", // Will not match regex!
			setup: func() {
				mockUserRepo.EXPECT().GetUserByID(ctx, uint64(1)).Return(origUser, nil)
			},
			wantErr:     true,
			errContains: "invalid email",
		},
		{
			name: "only phone update",
			userID: 1,
			newPhone: "12345",
			setup: func() {
				mockUserRepo.EXPECT().GetUserByID(ctx, uint64(1)).Return(origUser, nil)
				mockUserRepo.EXPECT().UpdateUser(ctx, gomock.AssignableToTypeOf(&dbmysql.User{})).DoAndReturn(func(_ context.Context, u *dbmysql.User) error {
					require.Equal(t, "12345", u.Phone)
					return nil
				})
			},
		},
		{
			name: "only profile update",
			userID: 1,
			newProfile: "bio X",
			setup: func() {
				mockUserRepo.EXPECT().GetUserByID(ctx, uint64(1)).Return(origUser, nil)
				mockUserRepo.EXPECT().UpdateUser(ctx, gomock.AssignableToTypeOf(&dbmysql.User{})).DoAndReturn(func(_ context.Context, u *dbmysql.User) error {
					require.Equal(t, "bio X", u.ProfileDetails)
					return nil
				})
			},
		},
		{
			name: "repo UpdateUser error",
			userID: 1,
			newEmail: "ok@b.com",
			setup: func() {
				mockUserRepo.EXPECT().GetUserByID(ctx, uint64(1)).Return(origUser, nil)
				mockUserRepo.EXPECT().UpdateUser(ctx, gomock.AssignableToTypeOf(&dbmysql.User{})).Return(errors.New("fail update"))
			},
			wantErr: true,
			errContains: "fail update",
		},


	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			err := svc.UpdateProfile(ctx, tc.userID, tc.newEmail, tc.newPhone, tc.newProfile)
			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// This test works for
// Can't friend yourself
// Already friends/pending not allowed
// Happy path
func TestUserService_SendFriendRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockUserRepo := NewMockUserRepository(ctrl)
	mockFriendRepo := NewMockFriendRepository(ctrl)
	mockDeviceRepo := NewMockDeviceRepository(ctrl)
	svc := NewUserService(mockUserRepo, mockFriendRepo, mockDeviceRepo)
	ctx := context.Background()

	tests := []struct {
		name        string
		userID      uint64
		targetID    uint64
		setup       func()
		wantErr     bool
		errContains string
	}{
		{
			name:        "cannot friend yourself",
			userID:      1,
			targetID:    1,
			setup:       func() {},
			wantErr:     true,
			errContains: "yourself",
		},
		{
			name:     "already friends",
			userID:   1,
			targetID: 2,
			setup: func() {
				mockFriendRepo.EXPECT().CheckFriendshipExists(ctx, uint64(1), uint64(2)).Return(true, nil)
			},
			wantErr:     true,
			errContains: "already friends",
		},
		{
			name:     "happy path",
			userID:   1,
			targetID: 2,
			setup: func() {
				mockFriendRepo.EXPECT().CheckFriendshipExists(ctx, uint64(1), uint64(2)).Return(false, nil)
				mockFriendRepo.EXPECT().CreateFriendRequest(ctx, gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "CheckFriendshipExists error",
			userID: 1, targetID: 2,
			setup: func() {
				mockFriendRepo.EXPECT().CheckFriendshipExists(ctx, uint64(1), uint64(2)).Return(false, errors.New("db fail"))
			},
			wantErr: true,
			errContains: "db fail",
		},
		{
			name: "CreateFriendRequest error",
			userID: 1, targetID: 2,
			setup: func() {
				mockFriendRepo.EXPECT().CheckFriendshipExists(ctx, uint64(1), uint64(2)).Return(false, nil)
				mockFriendRepo.EXPECT().CreateFriendRequest(ctx, gomock.Any()).Return(errors.New("db fail"))
			},
			wantErr: true,
			errContains: "db fail",
		},
		{
			name: "CheckFriendshipExists error",
			userID: 1, targetID: 2,
			setup: func() {
				mockFriendRepo.EXPECT().CheckFriendshipExists(ctx, uint64(1), uint64(2)).Return(false, errors.New("db fail"))
			},
			wantErr: true,
			errContains: "db fail",
		},
		{
			name: "CreateFriendRequest error",
			userID: 1, targetID: 2,
			setup: func() {
				mockFriendRepo.EXPECT().CheckFriendshipExists(ctx, uint64(1), uint64(2)).Return(false, nil)
				mockFriendRepo.EXPECT().CreateFriendRequest(ctx, gomock.Any()).Return(errors.New("db fail"))
			},
			wantErr: true,
			errContains: "db fail",
		},

	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			err := svc.SendFriendRequest(ctx, tc.userID, tc.targetID)
			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}



func TestUserService_AcceptFriendRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockUserRepo := NewMockUserRepository(ctrl)
	mockFriendRepo := NewMockFriendRepository(ctrl)
	mockDeviceRepo := NewMockDeviceRepository(ctrl)
	svc := NewUserService(mockUserRepo, mockFriendRepo, mockDeviceRepo)
	ctx := context.Background()

	pending := &dbmysql.Friend{UserID: 2, FriendUserID: 1, Status: "pending"}
	accepted := &dbmysql.Friend{UserID: 2, FriendUserID: 1, Status: "accepted"}
	tests := []struct {
		name        string
		userID      uint64
		requesterID uint64
		setup       func()
		wantErr     bool
		errContains string
	}{
		{
			name:   "pending request: success",
			userID: 1, requesterID: 2,
			setup: func() {
				// Friend request from 2->1, status pending
				mockFriendRepo.EXPECT().GetFriendRequest(ctx, uint64(2), uint64(1)).Return(pending, nil)
				mockFriendRepo.EXPECT().UpdateFriendRequest(ctx, gomock.AssignableToTypeOf(&dbmysql.Friend{})).Return(nil)
				mockFriendRepo.EXPECT().GetFriendRequest(ctx, uint64(1), uint64(2)).Return(nil, gorm.ErrRecordNotFound)
				mockFriendRepo.EXPECT().CreateFriendRequest(ctx, gomock.AssignableToTypeOf(&dbmysql.Friend{})).Return(nil)
			},
		},
		{
			name:   "no request found",
			userID: 1, requesterID: 3,
			setup: func() {
				mockFriendRepo.EXPECT().GetFriendRequest(ctx, uint64(3), uint64(1)).Return(nil, gorm.ErrRecordNotFound)
			},
			wantErr: true, errContains: "not found",
		},
		{
			name:   "already accepted",
			userID: 1, requesterID: 2,
			setup: func() {
				mockFriendRepo.EXPECT().GetFriendRequest(ctx, uint64(2), uint64(1)).Return(accepted, nil)
			},
			wantErr: true, errContains: "pending",
		},
		{
			name: "GetFriendRequest repo error",
			userID: 1, requesterID: 2,
			setup: func() {
				mockFriendRepo.EXPECT().GetFriendRequest(ctx, uint64(2), uint64(1)).Return(nil, errors.New("db err"))
			},
			wantErr: true,
			errContains: "db err",
		},
		{
			name: "UpdateFriendRequest repo error",
			userID: 1, requesterID: 2,
			setup: func() {
				mockFriendRepo.EXPECT().GetFriendRequest(ctx, uint64(2), uint64(1)).Return(&dbmysql.Friend{Status: "pending"}, nil)
				mockFriendRepo.EXPECT().UpdateFriendRequest(ctx, gomock.AssignableToTypeOf(&dbmysql.Friend{})).Return(errors.New("db error"))
			},
			wantErr: true, errContains: "db error",
		},
		{
			name: "CreateFriendRequest error (reverse)",
			userID: 1, requesterID: 2,
			setup: func() {
				mockFriendRepo.EXPECT().GetFriendRequest(ctx, uint64(2), uint64(1)).Return(&dbmysql.Friend{Status: "pending"}, nil)
				mockFriendRepo.EXPECT().UpdateFriendRequest(ctx, gomock.AssignableToTypeOf(&dbmysql.Friend{})).Return(nil)
				mockFriendRepo.EXPECT().GetFriendRequest(ctx, uint64(1), uint64(2)).Return(nil, gorm.ErrRecordNotFound)
				mockFriendRepo.EXPECT().CreateFriendRequest(ctx, gomock.AssignableToTypeOf(&dbmysql.Friend{})).Return(errors.New("fail create reverse"))
			},
			wantErr: true, errContains: "fail create reverse",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			err := svc.AcceptFriendRequest(ctx, tc.userID, tc.requesterID)
			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUserService_ListFriends(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockUserRepo := NewMockUserRepository(ctrl)
	mockFriendRepo := NewMockFriendRepository(ctrl)
	mockDeviceRepo := NewMockDeviceRepository(ctrl)
	svc := NewUserService(mockUserRepo, mockFriendRepo, mockDeviceRepo)
	ctx := context.Background()

	friends := []*dbmysql.User{
		{UserID: 2, Handle: "bob"},
		{UserID: 3, Handle: "eve"},
	}

	tests := []struct {
		name    string
		userID  uint64
		setup   func()
		wantErr bool
	}{
		{
			name:   "found friends",
			userID: 1,
			setup: func() {
				mockFriendRepo.EXPECT().ListFriends(ctx, uint64(1)).Return(friends, nil)
			},
		},
		{
			name:   "db error",
			userID: 1,
			setup: func() {
				mockFriendRepo.EXPECT().ListFriends(ctx, uint64(1)).Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
		
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			out, err := svc.ListFriends(ctx, tc.userID)
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, out)
			} else {
				require.NoError(t, err)
				require.Len(t, out, 2)
			}
		})
	}
}

func TestUserService_RegisterDevice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockUserRepo := NewMockUserRepository(ctrl)
	mockFriendRepo := NewMockFriendRepository(ctrl)
	mockDeviceRepo := NewMockDeviceRepository(ctrl)
	svc := NewUserService(mockUserRepo, mockFriendRepo, mockDeviceRepo)
	ctx := context.Background()

	tests := []struct {
		name        string
		userID      uint64
		token       string
		platform    string
		setup       func()
		wantErr     bool
		errContains string
	}{
		{
			name:   "success android",
			userID: 1, token: "fcm1", platform: "android",
			setup: func() {
				mockDeviceRepo.EXPECT().RegisterDevice(ctx, gomock.Any()).Return(nil)
			},
		},
		{
			name:   "missing token",
			userID: 1, token: "", platform: "android",
			setup:   func() {},
			wantErr: true, errContains: "device token",
		},
		{
			name:   "bad platform",
			userID: 1, token: "fcm2", platform: "windows",
			setup:   func() {},
			wantErr: true, errContains: "platform",
		},
		{
			name:   "repo error",
			userID: 1, token: "fcm3", platform: "android",
			setup: func() {
				mockDeviceRepo.EXPECT().RegisterDevice(ctx, gomock.Any()).Return(errors.New("oops"))
			},
			wantErr: true, errContains: "oops",
		},
		{
			name: "platform empty",
			userID: 1, token: "t1", platform: "",
			setup: func(){},
			wantErr: true, errContains: "platform",
		},


	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			err := svc.RegisterDevice(ctx, tc.userID, tc.token, tc.platform)
			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUserService_RemoveDevice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockUserRepo := NewMockUserRepository(ctrl)
	mockFriendRepo := NewMockFriendRepository(ctrl)
	mockDeviceRepo := NewMockDeviceRepository(ctrl)
	svc := NewUserService(mockUserRepo, mockFriendRepo, mockDeviceRepo)
	ctx := context.Background()

	tests := []struct {
		name    string
		token   string
		setup   func()
		wantErr bool
	}{
		{
			name:  "success",
			token: "fcm4",
			setup: func() {
				mockDeviceRepo.EXPECT().RemovedDevice(ctx, "fcm4").Return(nil)
			},
		},
		{
			name:  "repo error",
			token: "fcm4",
			setup: func() {
				mockDeviceRepo.EXPECT().RemovedDevice(ctx, "fcm4").Return(errors.New("delete err"))
			},
			wantErr: true,
		},
		
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			err := svc.RemoveDevice(ctx, tc.token)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUserService_GetUserDevices(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockUserRepo := NewMockUserRepository(ctrl)
	mockFriendRepo := NewMockFriendRepository(ctrl)
	mockDeviceRepo := NewMockDeviceRepository(ctrl)
	svc := NewUserService(mockUserRepo, mockFriendRepo, mockDeviceRepo)
	ctx := context.Background()

	sample := []*dbmysql.Device{
		{DeviceToken: "1", Platform: "android"},
		{DeviceToken: "2", Platform: "ios"},
	}

	tests := []struct {
		name    string
		userID  uint64
		setup   func()
		wantErr bool
	}{
		{
			name:   "success",
			userID: 1,
			setup: func() {
				mockDeviceRepo.EXPECT().GetUserDevices(ctx, uint64(1)).Return(sample, nil)
			},
		},
		{
			name:   "repo fail",
			userID: 2,
			setup: func() {
				mockDeviceRepo.EXPECT().GetUserDevices(ctx, uint64(2)).Return(nil, errors.New("fail devices"))
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			devices, err := svc.GetUserDevices(ctx, tc.userID)
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, devices)
			} else {
				require.NoError(t, err)
				require.NotNil(t, devices)
			}
		})
	}
}

func TestUserService_TouchDevice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockUserRepo := NewMockUserRepository(ctrl)
	mockFriendRepo := NewMockFriendRepository(ctrl)
	mockDeviceRepo := NewMockDeviceRepository(ctrl)
	svc := NewUserService(mockUserRepo, mockFriendRepo, mockDeviceRepo)
	ctx := context.Background()

	tests := []struct {
		name    string
		token   string
		setup   func()
		wantErr bool
	}{
		{
			name:  "success",
			token: "fcm8",
			setup: func() {
				mockDeviceRepo.EXPECT().UpdatedDeviceActivity(ctx, "fcm8").Return(nil)
			},
		},
		{
			name:  "fail",
			token: "fcm9",
			setup: func() {
				mockDeviceRepo.EXPECT().UpdatedDeviceActivity(ctx, "fcm9").Return(errors.New("not found"))
			},
			wantErr: true,
		},
		{
			name: "activity DB error",
			token: "failtoken",
			setup: func() {
				mockDeviceRepo.EXPECT().UpdatedDeviceActivity(ctx, "failtoken").Return(errors.New("db err"))
			},
			wantErr: true,
		},

	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			err := svc.TouchDevice(ctx, tc.token)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}


