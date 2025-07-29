package dbmysql

import (
	"time"
)

type Message struct {
	MessageID      uint      `gorm:"column:message_id;primaryKey;autoIncrement" json:"message_id"`
	ConversationID string    `gorm:"index;size:36" json:"conversation_id"`
	SenderID       string    `gorm:"index;size:36" json:"sender_id"`
	Content        string    `gorm:"type:text" json:"content"`
	SentAt         time.Time `gorm:"autoCreateTime" json:"sent_at"`
	Status         string    `gorm:"type:enum('delivered','read','deleted');default:'delivered'" json:"status"`
	MediaRefID     *uint     `gorm:"index"` // foreign key to media_refs
	//MediaRef       *MediaRef `gorm:"foreignKey:MediaRefID"` // eager load if needed
	//gorm.Model
}
