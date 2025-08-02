package dbmysql

// import (
// 	"GoSocial/internal/common"
// 	"GoSocial/internal/config"
// 	"context"
// 	"fmt"
// 	"time"

// 	"gorm.io/gorm"
// )

// func Create(id int64, fileID string, fileName string, contentType common.MediaFileType, size int64, uploadedBy string, uploadedAt time.Time, cnf config.Config) *MediaRef {
// 	url := fmt.Sprintf("%s%s", cnf.Server.MediaBaseURL, fileID)
// 	return &MediaRef{MediaRefID: id, FilePath: fileID, FileName: fileName, ContentType: contentType, Size: size, UploadedBy: uploadedBy, UploadedAt: uploadedAt, URL: url}
// }

// // ReadMedia fetches MediaRef metadata by ID
// func ReadMedia(db *gorm.DB, id uint) (*MediaRef, error) {
// 	var mediaRef MediaRef
// 	if err := db.First(&mediaRef, id).Error; err != nil {
// 		return nil, err
// 	}
// 	return &mediaRef, nil
// }

// // UpdateMedia updates MediaRef metadata (not file content)
// func UpdateMedia(db *gorm.DB, id uint, updates map[string]interface{}) error {
// 	return db.Model(&MediaRef{}).Where("media_ref_id = ?", id).Updates(updates).Error
// }

// // DeleteMedia deletes file from MongoDB and metadata from MySQL
// func DeleteMedia(db *gorm.DB, mediaStore interface {
// 	DeleteFile(ctx context.Context, fileID string) error
// }, id uint) error {
// 	var mediaRef MediaRef
// 	if err := db.First(&mediaRef, id).Error; err != nil {
// 		return err
// 	}
// 	// Delete from MongoDB
// 	if err := mediaStore.DeleteFile(context.Background(), mediaRef.FileID); err != nil {
// 		return err
// 	}
// 	// Delete from MySQL
// 	return db.Delete(&mediaRef).Error
// }
