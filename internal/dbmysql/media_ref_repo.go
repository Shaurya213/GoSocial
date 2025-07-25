package dbmysql

import (
	"fmt"
	"gosocial/internal/common"
	"gosocial/internal/config"
	"time"
)

func Create(id uint, fileID string, fileName string, contentType common.MediaFileType, size int64, uploadedBy string, uploadedAt time.Time, cnf config.Config) *MediaRef {
	url := fmt.Sprintf("%s%s", cnf.Server.MediaBaseURL, fileID)
	return &MediaRef{MediaRefID: id, FileID: fileID, FileName: fileName, ContentType: contentType, Size: size, UploadedBy: uploadedBy, UploadedAt: uploadedAt, URL: url}
}
