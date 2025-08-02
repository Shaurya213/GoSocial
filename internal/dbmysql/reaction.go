package dbmysql

import "time"

type Reaction struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	UserID    int64     `gorm:"column:user_id"`
	ContentID int64     `gorm:"column:content_id"`
	Type      string    `gorm:"column:type"` // like, love, laugh, etc.
	CreatedAt time.Time `gorm:"column:created_at"`

	user    User    `gorm:"foreignKey:UserID;references:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	content Content `gorm:"foreignKey:ContentID;references:ContentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
