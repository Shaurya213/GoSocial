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

// DetectFileType determines file type from MIME type
func DetectFileType(mimeType string) MediaFileType {
	if strings.HasPrefix(mimeType, "image/") {
		return MediaFileTypeImage
	}
	if strings.HasPrefix(mimeType, "video/") {
		return MediaFileTypeVideo
	}
	return MediaFileTypeImage // Default fallback
}
