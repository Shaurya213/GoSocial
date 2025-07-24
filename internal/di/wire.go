package di

import (
	"gosocial/internal/user"
	"github.com/google/wire"
	"gorm.io/gorm"
)


// Provider Functions from Repository layer
func NewUserRepo(db *gorm.DB) user.UserRepository {
	return user.NewUserRepository(db)
}

func NewFriendRepo(db *gorm.DB) user.FriendRepository {
	return user.NewFriendRepository(db)	
}

func NewDeviceRepo(db *gorm.DB) user.DeviceRepository {
	return user.NewDeviceRepository(db)
}

// Provider function from service layer
func NewUserService(userRepo user.UserRepository, friendRepo user.FriendRepository, deviceRepo user.DeviceRepository) user.UserService {
	return user.NewUserService(userRepo, friendRepo, deviceRepo)
}

//provider function from Handler
func NewHandler(userService user.UserService) *user.Handler {
	return user.NewHandler(userService)
}

// provider set- put all the provider functions from all the layer, 
// wire automatically sets all the wiring between provider functions
var UserSet = wire.NewSet( 
	NewUserRepo,
	NewFriendRepo,
	NewDeviceRepo,
	NewUserService,
	NewHandler,
)

// wire entry point
// it needs *user.Handler, so we are returning it with along all the helper or provider set
// giving or passing *gorm.DB do wire won't generate it
func InitializeUserHandler(db *gorm.DB) *user.Handler {
	wire.Build(UserSet)
	return &user.Handler{}
}
