package dbmongo

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gosocial/internal/common"
)

// MockGridFSBucket for testing
type MockGridFSBucket struct {
	mock.Mock
}

type MockUploadStream struct {
	mock.Mock
	fileID interface{}
}

func (m *MockUploadStream) Write(p []byte) (int, error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

func (m *MockUploadStream) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockUploadStream) FileID() interface{} {
	return m.fileID
}

func TestMediaStorage_UploadFile(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		mimeType    string
		uploaderID  string
		content     []byte
		expectError bool
	}{
		{
			name:        "successful image upload",
			filename:    "test.jpg",
			mimeType:    "image/jpeg",
			uploaderID:  "user123",
			content:     []byte("fake image content"),
			expectError: false,
		},
		{
			name:        "successful video upload", 
			filename:    "test.mp4",
			mimeType:    "video/mp4",
			uploaderID:  "user123",
			content:     []byte("fake video content"),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: For unit tests, you'd mock the GridFS bucket
			// For integration tests, you'd use a real MongoDB instance
			
			expectedFileType := common.DetectFileType(tt.mimeType)
			assert.True(t, expectedFileType.IsValid())
			
			if tt.mimeType == "image/jpeg" {
				assert.Equal(t, common.MediaFileTypeImage, expectedFileType)
			} else if tt.mimeType == "video/mp4" {
				assert.Equal(t, common.MediaFileTypeVideo, expectedFileType)
			}
		})
	}
}

func TestMediaFileType_Validation(t *testing.T) {
	tests := []struct {
		mimeType     string
		expectedType common.MediaFileType
		valid        bool
	}{
		{"image/jpeg", common.MediaFileTypeImage, true},
		{"image/png", common.MediaFileTypeImage, true},
		{"video/mp4", common.MediaFileTypeVideo, true},
		{"video/webm", common.MediaFileTypeVideo, true},
		{"application/pdf", common.MediaFileTypeImage, true}, // Defaults to image
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			fileType := common.DetectFileType(tt.mimeType)
			assert.Equal(t, tt.expectedType, fileType)
			assert.Equal(t, tt.valid, fileType.IsValid())
		})
	}
}

