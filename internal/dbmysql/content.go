package dbmysql

import (
	"time"
)

// content.go
type Content struct {
	ContentID   int64      `gorm:"primaryKey;autoIncrement;column:content_id"`
	AuthorID    int64      `gorm:"column:author_id"`
	Type        string     `gorm:"type:ENUM('POST','STORY','REEL');column:type"`
	TextContent *string    `gorm:"column:text_content"`
	MediaRefID  *int64     `gorm:"column:media_ref_id"`
	Privacy     string     `gorm:"type:ENUM('public','friends','private');column:privacy"`
	Expiration  *time.Time `gorm:"column:expiration"`
	Duration    *int       `gorm:"column:duration"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`

	User     User     `gorm:"foreignKey:AuthorID"`
	MediaRef MediaRef `gorm:"references:MediaRefID"` // no foreignKey here, fk is inferred from MediaRefID field
}
