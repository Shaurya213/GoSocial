package dbmysql

import (
	"time"
	"gorm.io/gorm"
)

type Message struct {
	ID             uint   `gorm:"primaryKey"`
	ConversationID string `gorm:"index;size:36"`
	SenderID       string `gorm:"index;size:36"`
	Content        string `gorm:"type:text"`
	SentAt         time.Time
	gorm.Model
}

func (Message) TableName() string {
	return "messages"
}
