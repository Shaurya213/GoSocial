package dbmysql

import "time"

type MediaRef struct {
	MediaRefID int64     `gorm:"primaryKey;autoIncrement;column:media_ref_id"`
	Type       string    `gorm:"type:ENUM('image','video');column:type"` // media type
	FilePath   string    `gorm:"column:file_path"`                       // GridFS ObjectID or S3 path
	FileName   string    `gorm:"column:file_name"`                       // original uploaded name
	UploadedBy int64     `gorm:"column:uploaded_by"`                     // FK to users.id
	UploadedAt time.Time `gorm:"column:uploaded_at"`                     // time of upload
	SizeBytes  int64     `gorm:"column:size_bytes"`                      // file size in bytes

	UploadedByUser User `gorm:"foreignKey:UploadedBy;references:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"uploaded_by_user"`
}
