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

func InitChatHandler(db *gorm.DB) *handler.ChatHandler {
	wire.Build(
		repository.NewChatRepository,
		service.NewChatService,
		handler.NewChatHandler,
	)
	return &handler.ChatHandler{} 
}
