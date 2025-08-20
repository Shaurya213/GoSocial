package shared

import (
	"gosocial/internal/common"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMediaFileType_String(t *testing.T) {
	assert.Equal(t, "image", common.MediaFileTypeImage.String())
	assert.Equal(t, "video", common.MediaFileTypeVideo.String())
}

func TestMediaFileType_IsValid(t *testing.T) {
	assert.True(t, common.MediaFileTypeImage.IsValid())
	assert.True(t, common.MediaFileTypeVideo.IsValid())

	// Test invalid type
	invalidType := common.MediaFileType("invalid")
	assert.False(t, invalidType.IsValid())
}

func TestDetectFileType_Images(t *testing.T) {
	imageTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"image/webp",
	}

	for _, mimeType := range imageTypes {
		result := common.DetectFileType(mimeType)
		assert.Equal(t, common.MediaFileTypeImage, result, "Failed for MIME type: %s", mimeType)
	}
}

func TestDetectFileType_Videos(t *testing.T) {
	videoTypes := []string{
		"video/mp4",
		"video/avi",
		"video/mov",
		"video/webm",
		"video/mkv",
	}

	for _, mimeType := range videoTypes {
		result := common.DetectFileType(mimeType)
		assert.Equal(t, common.MediaFileTypeVideo, result, "Failed for MIME type: %s", mimeType)
	}
}

func TestDetectFileType_DefaultFallback(t *testing.T) {
	unknownTypes := []string{
		"application/pdf",
		"text/plain",
		"audio/mp3",
		"unknown/type",
		"",
	}

	for _, mimeType := range unknownTypes {
		result := common.DetectFileType(mimeType)
		assert.Equal(t, common.MediaFileTypeImage, result, "Failed for MIME type: %s", mimeType)
	}
}

func TestDetectFileType_EdgeCases(t *testing.T) {
	edgeCases := []struct {
		input    string
		expected common.MediaFileType
	}{
		{"IMAGE/JPEG", common.MediaFileTypeImage}, // Case insensitive
		{"Video/MP4", common.MediaFileTypeVideo},  // Case insensitive
		{"image/", common.MediaFileTypeImage},     // Partial match
		{"video/", common.MediaFileTypeVideo},     // Partial match
	}

	for _, testCase := range edgeCases {
		result := common.DetectFileType(testCase.input)
		assert.Equal(t, testCase.expected, result, "Failed for input: %s", testCase.input)
	}
}
