package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMediaFileType_String(t *testing.T) {
	assert.Equal(t, "image", MediaFileTypeImage.String())
	assert.Equal(t, "video", MediaFileTypeVideo.String())
}

func TestMediaFileType_IsValid(t *testing.T) {
	assert.True(t, MediaFileTypeImage.IsValid())
	assert.True(t, MediaFileTypeVideo.IsValid())
	
	// Test invalid type
	invalidType := MediaFileType("invalid")
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
		result := DetectFileType(mimeType)
		assert.Equal(t, MediaFileTypeImage, result, "Failed for MIME type: %s", mimeType)
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
		result := DetectFileType(mimeType)
		assert.Equal(t, MediaFileTypeVideo, result, "Failed for MIME type: %s", mimeType)
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
		result := DetectFileType(mimeType)
		assert.Equal(t, MediaFileTypeImage, result, "Failed for MIME type: %s", mimeType)
	}
}

func TestDetectFileType_EdgeCases(t *testing.T) {
	edgeCases := []struct {
		input    string
		expected MediaFileType
	}{
		{"IMAGE/JPEG", MediaFileTypeImage}, // Case insensitive
		{"Video/MP4", MediaFileTypeVideo},   // Case insensitive
		{"image/", MediaFileTypeImage},      // Partial match
		{"video/", MediaFileTypeVideo},      // Partial match
	}
	
	for _, testCase := range edgeCases {
		result := DetectFileType(testCase.input)
		assert.Equal(t, testCase.expected, result, "Failed for input: %s", testCase.input)
	}
}

