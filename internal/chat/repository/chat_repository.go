package repository

import (
	"context"
	"gorm.io/gorm"
	_ "gorm.io/gorm"
	//"gosocial/internal/chat/models"
	"GoSocial/internal/dbmysql"
)

type ChatRepository interface {
	Save(ctx context.Context, msg *dbmysql.Message) error
	FetchHistory(ctx context.Context, conversationID string) ([]*dbmysql.Message, error)
}

type chatRepo struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepo{
		db: db,
	}
}

func (r *chatRepo) Save(ctx context.Context, msg *dbmysql.Message) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

func (r *chatRepo) FetchHistory(ctx context.Context, conversationID string) ([]*dbmysql.Message, error) {
	var messages []*dbmysql.Message
	return messages, r.db.Where("conversation_id = ?", conversationID).Find(&messages).Error
}
