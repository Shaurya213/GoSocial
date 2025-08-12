package common

import "strings"

// MediaFileType represents file format types from MediaRef table schema (PDF)
type MediaFileType string

const (
	MediaFileTypeImage MediaFileType = "image"
	MediaFileTypeVideo MediaFileType = "video"
)

// String returns the string representation
func (mft MediaFileType) String() string {
	return string(mft)
}

// IsValid checks if the media file type is valid
func (mft MediaFileType) IsValid() bool {
	return mft == MediaFileTypeImage || mft == MediaFileTypeVideo
}

func DetectFileType(mimeType string) MediaFileType {
	lowerMimeType := strings.ToLower(mimeType)
	if strings.HasPrefix(lowerMimeType, "image/") {
		return MediaFileTypeImage
	}
	if strings.HasPrefix(lowerMimeType, "video/") {
		return MediaFileTypeVideo
	}
	return MediaFileTypeImage // Default fallback
}

