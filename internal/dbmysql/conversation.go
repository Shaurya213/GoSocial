package dbmysql

import (
	"gorm.io/gorm"
)

type Conversation struct {
	ID string `gorm:"primaryKey;size:36"`
	ParticipantsIDs string `gorm:"type:json"`
	gorm.Model
}
