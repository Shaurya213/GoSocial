package dbmysql

import (
	"time"
)

type Message struct {
	ID             uint   `gorm:"primaryKey"`
	ConversationID string `gorm:"index;size:36"`
	SenderID       string `gorm:"index;size:36"`
	content        string `gorm:"type:text"`
	SentAt         time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
