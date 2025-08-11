package dbmysql
import "time"

type Content struct {
	ContentID   uint64     `gorm:"primaryKey;autoIncrement" json:"content_id"`
	AuthorID    uint64     `gorm:"index;not null" json:"author_id"`
	Type        string     `gorm:"type:enum('POST','STORY','REEL');not null" json:"type"`
	TextContent string     `gorm:"type:text" json:"text_content"`
	MediaRefID  *uint64    `gorm:"index" json:"media_ref_id"` // Nullable
	Privacy     string     `gorm:"type:enum('public','friends','private');not null" json:"privacy"`
	Expiration  *time.Time `json:"expiration"` // for stories
	Duration    *int       `json:"duration"`   // for reels
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}
