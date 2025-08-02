package dbmysql

import "time"

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

	user     User     `gorm:"foreignKey:AuthorID;references:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	MediaRef MediaRef `gorm:"foreignKey:MediaRefID;references:MediaRefID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}
