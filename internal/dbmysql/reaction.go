package dbmysql

import "time"

type Reaction struct {
	ID        int64     `gorm:"primaryKey;column:id"`
	UserID    int64     `gorm:"column:user_id"`
	ContentID int64     `gorm:"column:content_id"`
	Type      string    `gorm:"column:type"` // like, love, laugh, etc.
	CreatedAt time.Time `gorm:"column:created_at"`
}
