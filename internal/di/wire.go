//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"
	"gorm.io/gorm"

	"gosocial/internal/chat/handler"
	"gosocial/internal/chat/repository"
	"gosocial/internal/chat/service"
)

// This is just a declaration — wire will generate the real body
func InitChatHandler(db *gorm.DB) *handler.ChatHandler {
	wire.Build(
		repository.NewChatRepository,
		service.NewChatService,
		handler.NewChatHandler,
	)
	return &handler.ChatHandler{} // dummy for compilation
}
