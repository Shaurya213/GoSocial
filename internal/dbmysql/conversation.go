package dbmysql

import (
	"gorm.io/gorm"
	"time"
)

type Conversation struct {
	ConversationID  string    `gorm:"primaryKey;size:36"`
	ParticipantsIDs string    `gorm:"type:json" json:"participant_ids"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	gorm.Model                //This also will by default include CreatedAt UpdatedAt DeletedAt etc. but we specify
}
