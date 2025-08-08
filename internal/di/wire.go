//go:build wireinject
// +build wireinject

package di

import (
	"gosocial/internal/chat/handler"
	"gosocial/internal/chat/repository"
	"gosocial/internal/chat/service"
	"gosocial/internal/dbmysql"
	"gosocial/internal/config"

	"github.com/google/wire"
)

// ChatProviderSet contains all providers for chat service
var ChatProviderSet = wire.NewSet(
	config.LoadConfig,
	// Use existing database connection function
	dbmysql.NewMySQL,
	
	// Repository layer
	repository.NewChatRepository,
	
	// Service layer
	service.NewChatService,
	
	// Handler layer
	handler.NewChatHandler,
)

// InitializeChatService wires up all dependencies for the chat service
func InitializeChatService() (*handler.ChatHandler, func(), error) {
	wire.Build(ChatProviderSet)
	return nil, nil, nil
} 

