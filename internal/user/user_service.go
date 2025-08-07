package user

import (
	"context"
	"errors"
	"gosocial/internal/common"
	"gosocial/internal/dbmysql"
	"time"
	"gorm.io/gorm"
)

type UserService interface {
	RegisterUser(ctx context.Context, handle, email, password string) (*dbmysql.User, string, error)
	LoginUser(ctx context.Context, handle, password string) (*dbmysql.User, string, error)
	GetProfile(ctx context.Context, userID uint64) (*dbmysql.User, error)
	UpdateProfile(ctx context.Context, userID uint64, email, phone, profileDetails string) error
	SendFriendRequest(ctx context.Context, userID, targetUserID uint64) error
	AcceptFriendRequest(ctx context.Context, userID, requesterID uint64) error
	ListFriends(ctx context.Context, userID uint64) ([]*dbmysql.User, error)
	RegisterDevice(ctx context.Context, userID uint64, token, platform string) error
	RemoveDevice(ctx context.Context, token string) error
	GetUserDevices(ctx context.Context, userID uint64) ([]*dbmysql.Device, error)
	TouchDevice(ctx context.Context, token string) error
}

type userService struct {
	userRepo 	UserRepository
	friendRepo 	FriendRepository
	deviceRepo 	DeviceRepository
}

func NewUserService(userRepo UserRepository, friendRepo FriendRepository, deviceRepo DeviceRepository) UserService {
    return &userService{userRepo: userRepo, friendRepo: friendRepo, deviceRepo: deviceRepo}
}



func(s *userService) RegisterUser(ctx context.Context, handle, email, password string) (*dbmysql.User, string, error) {
	//validating handle
	if err := common.ValidateHandle(handle); err!= nil {
		return nil, "", err
	}

	if err := common.ValidateEmail(email); err!= nil {
		return nil, "", err
	}

	if err := common.ValidatePassword(password); err != nil {
		return nil, "", err
	}

	//duplicates check
	exists, err := s.userRepo.CheckUserExists(ctx, handle)
	if err != nil {
		return nil, "", err
	}
	if exists {
		return nil, "", errors.New("handle already exists")
	}

	//hashing password
	hashed, err := common.HashPassword(password)
	if err != nil {
		return nil, "", err
	}

	//create user
	user := &dbmysql.User{
		Handle: handle,
		Email: email,
		PasswordHash: hashed,
		Status: "active",
	}

	err = s.userRepo.CreateUser(ctx, user)

	if err != nil {
		return nil, "", err
	}

	//jwt
	token, err := common.GenerateToken(user.UserID, user.Handle)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil

}


func (s *userService) LoginUser(ctx context.Context, handle, password string)(*dbmysql.User, string, error) {
	if handle == "" || password == "" {
		return nil, "", errors.New("handle and password required")
	}

	user, err := s.userRepo.GetUserByHandle(ctx, handle)
	if err != nil {
		return nil, "", err
	}

	if user.Status != "active" {
		return nil, "", errors.New("user is not active")
	}

	if err:= common.CheckPassword(password, user.PasswordHash); err != nil {
		return nil, "", errors.New("invalid password")
	}

	token, err := common.GenerateToken(user.UserID, user.Handle)
	if err != nil{
		return nil, "", err
	}
	return user, token, nil
}


func (s *userService) GetProfile(ctx context.Context, userID uint64)(*dbmysql.User, error) {
	return s.userRepo.GetUserByID(ctx, userID)
}

func (s *userService) UpdateProfile(ctx context.Context, userID uint64, email, phone, profileDetails string) error {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil{
		return err
	}
	if email != ""{
		if err := common.ValidateEmail(email); err != nil {
			return err
		}
		user.Email = email
	}

	if phone != ""{
		user.Phone = phone
	}

	if profileDetails != "" {
		user.ProfileDetails = profileDetails
	}

	return s.userRepo.UpdateUser(ctx, user)
}


func (s *userService) SendFriendRequest(ctx context.Context, userID, targetUserID uint64) error {
	if userID == targetUserID {
		return errors.New("cannot send rquest to yourself")
	}

	alreadyFriends, err := s.friendRepo.CheckFriendshipExists(ctx, userID, targetUserID)
	if err != nil {
		return err
	}

	if alreadyFriends {
		return errors.New("already friends or request pending")
	}

	friend := &dbmysql.Friend{
		UserID: userID,
		FriendUserID: targetUserID,
		Status: "pending",
	}

	return s.friendRepo.CreateFriendRequest(ctx, friend)
}

func (s *userService) AcceptFriendRequest(ctx context.Context, userID, requesterID uint64) error {
    friendReq, err := s.friendRepo.GetFriendRequest(ctx, requesterID, userID)
    if err != nil {
        return err
    }
    if friendReq.Status != "pending" {
        return errors.New("friend request is not pending")
    }
    now := time.Now()
    friendReq.Status = "accepted"
    friendReq.AcceptedAt = &now
    if err := s.friendRepo.UpdateFriendRequest(ctx, friendReq); err!= nil{
        return err
    }
    
    already, err := s.friendRepo.GetFriendRequest(ctx, userID, requesterID)
    if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
        return err
    }
    if already == nil {
        reverse := &dbmysql.Friend{
            UserID: userID,
            FriendUserID: requesterID,
            Status: "accepted",
            AcceptedAt: &now,
        }
        if err := s.friendRepo.CreateFriendRequest(ctx, reverse); err != nil {
            return err
        }
    }
    return nil
}


func (s *userService) ListFriends(ctx context.Context, userID uint64)([]*dbmysql.User, error) {
	return s.friendRepo.ListFriends(ctx, userID)
}


func( s *userService) RegisterDevice(ctx context.Context, userID uint64, token, platform string) error {
	if token == "" {
		return errors.New("device token required")
	}
	if platform != "android" && platform != "ios" && platform != "web" {
        return errors.New("invalid platform")
    }
	
	device := &dbmysql.Device{
        DeviceToken:  token,
        UserID:       userID,
        Platform:     platform,
        RegisteredAt: time.Now(),
        LastActive:   time.Now(),
    }

	return s.deviceRepo.RegisterDevice(ctx, device)
}

func(s *userService) RemoveDevice(ctx context.Context, token string) error {
	return s.deviceRepo.RemoveDevice(ctx, token)
}

func(s *userService) GetUserDevices(ctx context.Context, userID uint64)([]*dbmysql.Device, error) {
	return s.deviceRepo.GetUserDevices(ctx, userID)
}

func(s *userService) TouchDevice(ctx context.Context, token string) error {
	return s.deviceRepo.UpdateDeviceActivity(ctx, token)
}