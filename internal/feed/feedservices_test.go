package feed_test

import (
	"context"
	"testing"

	userpb "GoSocial/api/v1/user" // adjust this to your proto import path
	"GoSocial/internal/feed"
	"GoSocial/internal/feed/mocks" // import path for generated mocks
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestGetUserFriendIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserClient := mocks.NewMockUserServiceClient(ctrl)
	service := feed.FeedService{UserClient: mockUserClient}

	ctx := context.Background()
	userID := int64(101)

	mockResponse := &userpb.FriendList{
		Friends: []*userpb.Friend{
			{UserId: 201, Handle: "alice"},
			{UserId: 202, Handle: "bob"},
		},
	}

	mockUserClient.EXPECT().
		ListFriends(ctx, &userpb.UserID{UserId: userID}).
		Return(mockResponse, nil)

	friendIDs, err := service.GetUserFriendIDs(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, []int64{201, 202}, friendIDs)
}
