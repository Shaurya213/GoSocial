package dbmysql

import "time"

type Reaction struct {
	ReactionID   uint64    `gorm:"primaryKey;autoIncrement" json:"reaction_id"`
	UserID       uint64    `gorm:"index;not null" json:"user_id"`
	ContentID    uint64    `gorm:"index;not null" json:"content_id"`
	ReactionType string    `gorm:"type:enum('like','love');not null" json:"reaction_type"` // like,love are the reaction type
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}
