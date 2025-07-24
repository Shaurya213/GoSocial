package user

import (
	"GoSocial/internal/dbmysql"
	"context"

	"gorm.io/gorm"
)

type FriendRepository interface {
	CreateFriendRequest(ctx context.Context, friend *dbmysql.Friend) error
	GetFriendRequest(ctx context.Context, userID, friendUserID uint64) (*dbmysql.Friend, error)
	UpdateFriendRequest(ctx context.Context, friend *dbmysql.Friend) error
	ListFriends(ctx context.Context, userID uint64) ([]*dbmysql.User, error)
	ListPendingRequests(ctx context.Context, userID uint64) ([]*dbmysql.Friend, error)
	CheckFriendshipExists(ctx context.Context, userID, friendUserID uint64) (bool, error)
}

type friendRepository struct {
	db *gorm.DB
}

func NewFriendRepository(db *gorm.DB) FriendRepository {
	return &friendRepository{db: db}
}

func (r *friendRepository) CreateFriendRequest(ctx context.Context, friend *dbmysql.Friend) error {
	return r.db.WithContext(ctx).Create(friend).Error
}

func (r *friendRepository) GetFriendRequest(ctx context.Context, userID, friendUserID uint64) (*dbmysql.Friend, error) {
	var friend dbmysql.Friend
	err := r.db.WithContext(ctx).Where("user_id = ? AND friend_user_id = ?", userID, friendUserID).First(&friend).Error
	if err != nil {
		return nil, err
	}
	return &friend, nil
}

func (r *friendRepository) UpdateFriendRequest(ctx context.Context, friend *dbmysql.Friend) error {
	return r.db.WithContext(ctx).Save(friend).Error
}

// func (r *friendRepository) ListFriends(ctx context.Context, userID uint64) ([]*dbmysql.Friend, error) {
// 	var friends []*dbmysql.Friend
// 	err := r.db.WithContext(ctx).
// 		Where("user_id = ? AND status = ?", userID, "accepted").
// 		Preload("FriendUSer").
// 		Order("accepted_at DESC").
// 		Find(&friends).Error
// 	return friends, err
// }

func (r *friendRepository) ListFriends(ctx context.Context, userID uint64) ([]*dbmysql.User, error) {
	var friends []dbmysql.Friend

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, "accepted").
		Preload("FriendUser").
		Order("accepted_at DESC").
		Find(&friends).Error

	if err != nil {
		return nil, err
	}

	var friendUsers []*dbmysql.User
	for _, f := range friends {
		friendUsers = append(friendUsers, &f.FriendUser)
	}

	return friendUsers, nil
}

func (r *friendRepository) ListPendingRequests(ctx context.Context, userID uint64) ([]*dbmysql.Friend, error) {
	var requests []*dbmysql.Friend
	err := r.db.WithContext(ctx).
		Where("friend_user_id = ? AND status = ?", userID, "pending").
		Preload("User").
		Order("requested_at DESC").
		Find(&requests).Error
	return requests, err
}

func (r *friendRepository) CheckFriendshipExists(ctx context.Context, userID, friendUserID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&dbmysql.Friend{}).
		Where("(user_id = ? AND friend_user_id = ?) OR (user_id = ? AND friend_user_id = ?)", userID, friendUserID, friendUserID, userID).
		Count(&count).Error
	return count > 0, err
}
