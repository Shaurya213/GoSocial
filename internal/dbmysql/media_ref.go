package dbmysql

import (
    "gorm.io/gorm"
)

type MediaRef struct {
    ID          uint   `gorm:"primaryKey" json:"id"`
    FileID      string `gorm:"size:24;uniqueIndex" json:"file_id"` 
    Type        string `gorm:"size:20" json:"type"`
    FileName    string `gorm:"size:255" json:"file_name"`
    ContentType string `gorm:"size:100" json:"content_type"`
    URL         string `gorm:"size:500" json:"url"`
    Size        int64  `json:"size"`
    UploadedBy  string `gorm:"size:36;index" json:"uploaded_by"` 
    gorm.Model  
}

func (MediaRef) TableName() string {
    return "media_refs"
}
