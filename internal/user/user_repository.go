package user

import (
	"GoSocial/internal/dbmysql"
	"context"
	"gorm.io/gorm"
)

// in interface, all the merhods related to user is written, and implemented
type UserRepository interface {
	//for some basic crud, like create, getuserbyid, getuserbyprofile, updateuser
	CreateUser(ctx context.Context, user *dbmysql.User) error
	GetUserByID(ctx context.Context, userID uint64) (*dbmysql.User, error)
	GetUserByHandle(ctx context.Context, handle string) (*dbmysql.User, error)
	UpdateUser(ctx context.Context, user *dbmysql.User) error

	GetUserByEmail(ctx context.Context, email string) (*dbmysql.User, error)
	CheckUserExists(ctx context.Context, handle string) (bool, error)
}

// this struct implements above struct
type userRepository struct {
	db *gorm.DB //*gorm.DB points to the DB connection
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) CreateUser(ctx context.Context, user *dbmysql.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) GetUserByID(ctx context.Context, userID uint64) (*dbmysql.User, error) {
	var user dbmysql.User
	err := r.db.WithContext(ctx).Where("user_id = ? AND status = ?", userID, "active").First(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) GetUserByHandle(ctx context.Context, handle string) (*dbmysql.User, error) {
	var user dbmysql.User
	err := r.db.WithContext(ctx).Where("handle = ? AND status = ?", handle, "active").First(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, user *dbmysql.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*dbmysql.User, error) {
	var user dbmysql.User
	err := r.db.WithContext(ctx).Where("email = ? AND status = ?", email, "active").First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) CheckUserExists(ctx context.Context, handle string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&dbmysql.User{}).Where("handle = ?", handle).Count(&count).Error
	return count > 0, err
}
