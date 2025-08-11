package repository

import (
	"context"
	"gorm.io/gorm"
	
	"log"

	//"gosocial/internal/chat/models"
	"gosocial/internal/dbmysql"
)


type ChatRepository interface {
	Save(ctx context.Context, msg *dbmysql.Message) error
	FetchHistory(ctx context.Context, conversationID string) ([]*dbmysql.Message, error)
}

type chatRepo struct {
	migrated bool
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepo{
		db: db,
		migrated: false,
	}
}


func (r *chatRepo) Save(ctx context.Context, msg *dbmysql.Message) error {
	if !r.migrated {
		r.autoMigrate()
		return r.db.WithContext(ctx).Create(msg).Error
	} else {
		return r.db.WithContext(ctx).Create(msg).Error
	}
}

func (r *chatRepo) FetchHistory(ctx context.Context, conversationID string) ([]*dbmysql.Message, error) {
	if !r.migrated {
		r.autoMigrate()
		var messages []*dbmysql.Message
		return messages, r.db.Where("conversation_id = ?", conversationID).Find(&messages).Error
	} else {
		var messages []*dbmysql.Message
		return messages, r.db.Where("conversation_id = ?", conversationID).Find(&messages).Error
	}
}
func (r *chatRepo)autoMigrate() {
	if !r.migrated {
		if err := r.db.AutoMigrate(&dbmysql.Message{}); err != nil {
			log.Fatalf("Failed to migrate database: %v", err)
		}
	}
}
