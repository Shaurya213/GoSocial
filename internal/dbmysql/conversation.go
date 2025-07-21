package dbmysql

import (
	"time"
)

type Conversation struct {
	ID string `gorm:"primaryKey;size:36"`
	ParticipantsIDs string `gorm:"type:json"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
