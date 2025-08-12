package dbmongo

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gosocial/internal/common"
	"gosocial/internal/config"
)

var testConfig *config.Config

func TestMain(m *testing.M) {
	// Use existing MongoDB from docker-compose instead of testcontainers
	testConfig = &config.Config{
		MongoDB: config.MongoDBConfig{
			Host:     getEnvOrDefault("MONGO_HOST", "localhost"),
			Port:     getEnvOrDefault("MONGO_PORT", "27017"), 
			Username: getEnvOrDefault("MONGO_USERNAME", "admin"),
			Password: getEnvOrDefault("MONGO_PASSWORD", "admin123"),
			Database: getEnvOrDefault("MONGO_DATABASE", "gosocial_test"), // Use test database
		},
	}

	// Run tests
	code := m.Run()
	os.Exit(code)
}

func TestMongoConnection_Integration(t *testing.T) {
	ctx := context.Background()
	
	// Test actual MongoDB connection to your docker-compose setup
	client, err := NewMongoConnection(testConfig)
	require.NoError(t, err, "Failed to connect to existing MongoDB - ensure docker-compose is running")
	defer client.Close(ctx)
	
	// Verify connection works
	err = client.Client.Ping(ctx, nil)
	assert.NoError(t, err)
	
	// Verify GridFS bucket is created
	assert.NotNil(t, client.GridFS)
	assert.NotNil(t, client.Database)
}

func TestMediaStorage_WithExistingMongoDB(t *testing.T) {
	ctx := context.Background()
	
	// Connect to your existing MongoDB from docker-compose
	client, err := NewMongoConnection(testConfig)
	require.NoError(t, err, "Ensure MongoDB is running: docker-compose -f docker-compose-mongo.yml up -d")
	defer client.Close(ctx)
	
	storage := NewMediaStorage(client)
	
	t.Run("upload_and_download_file", func(t *testing.T) {
		// Test file content
		testContent := "This is test file content for GridFS upload"
		reader := strings.NewReader(testContent)
		
		// Upload file
		uploaded, err := storage.UploadFile(ctx, "test.txt", "text/plain", "user123", reader)
		
		assert.NoError(t, err)
		assert.NotEmpty(t, uploaded.ID)
		assert.Equal(t, "test.txt", uploaded.Filename)
		assert.Equal(t, int64(len(testContent)), uploaded.Size)
		assert.Equal(t, common.MediaFileTypeImage, uploaded.FileType) // text defaults to image
		assert.Equal(t, "user123", uploaded.UploadedBy)
		
		// Download file
		downloadReader, mediaFile, err := storage.DownloadFile(ctx, uploaded.ID)
		assert.NoError(t, err)
		assert.Equal(t, "test.txt", mediaFile.Filename)
		assert.Equal(t, "user123", mediaFile.UploadedBy)
		
		// Read downloaded content
		downloadedContent, err := io.ReadAll(downloadReader)
		assert.NoError(t, err)
		assert.Equal(t, testContent, string(downloadedContent))
		
		// Clean up - delete the test file
		err = storage.DeleteFile(ctx, uploaded.ID)
		assert.NoError(t, err)
	})
	
	t.Run("upload_image_file", func(t *testing.T) {
		imageContent := "fake-image-content"
		reader := strings.NewReader(imageContent)
		
		uploaded, err := storage.UploadFile(ctx, "image.jpg", "image/jpeg", "user456", reader)
		
		assert.NoError(t, err)
		assert.Equal(t, "image.jpg", uploaded.Filename)
		assert.Equal(t, common.MediaFileTypeImage, uploaded.FileType)
		assert.Equal(t, "user456", uploaded.UploadedBy)
		
		// Clean up
		storage.DeleteFile(ctx, uploaded.ID)
	})
	
	t.Run("upload_video_file", func(t *testing.T) {
		videoContent := "fake-video-content"
		reader := strings.NewReader(videoContent)
		
		uploaded, err := storage.UploadFile(ctx, "video.mp4", "video/mp4", "user789", reader)
		
		assert.NoError(t, err)
		assert.Equal(t, "video.mp4", uploaded.Filename)
		assert.Equal(t, common.MediaFileTypeVideo, uploaded.FileType)
		assert.Equal(t, "user789", uploaded.UploadedBy)
		
		// Clean up
		storage.DeleteFile(ctx, uploaded.ID)
	})
	
	t.Run("delete_file", func(t *testing.T) {
		// Upload a file to delete
		content := strings.NewReader("content to be deleted")
		uploaded, err := storage.UploadFile(ctx, "delete-me.txt", "text/plain", "user123", content)
		require.NoError(t, err)
		
		// Delete the file
		err = storage.DeleteFile(ctx, uploaded.ID)
		assert.NoError(t, err)
		
		// Verify file is deleted (should get error when trying to download)
		_, _, err = storage.DownloadFile(ctx, uploaded.ID)
		assert.Error(t, err)
	})
	
	t.Run("download_nonexistent_file", func(t *testing.T) {
		// Try to download non-existent file
		_, _, err := storage.DownloadFile(ctx, "507f1f77bcf86cd799439011") // Valid ObjectID format
		assert.Error(t, err)
	})
	
	t.Run("delete_nonexistent_file", func(t *testing.T) {
		// Try to delete non-existent file
		err := storage.DeleteFile(ctx, "507f1f77bcf86cd799439011") // Valid ObjectID format
		assert.Error(t, err)
	})
	
	t.Run("invalid_objectid_handling", func(t *testing.T) {
		// Test invalid ObjectID
		_, _, err := storage.DownloadFile(ctx, "invalid-objectid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid file ID")
		
		err = storage.DeleteFile(ctx, "invalid-objectid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid file ID")
	})
}

func TestGetStringFromMap_Integration(t *testing.T) {
	// Test the helper function with various scenarios
	testMap := map[string]interface{}{
		"string_key": "string_value",
		"int_key":    123,
		"bool_key":   true,
		"nil_key":    nil,
	}
	
	tests := []struct {
		name     string
		input    map[string]interface{}
		key      string
		expected string
	}{
		{"valid_string", testMap, "string_key", "string_value"},
		{"non_string_value", testMap, "int_key", ""},
		{"nil_value", testMap, "nil_key", ""},
		{"missing_key", testMap, "missing", ""},
		{"nil_map", nil, "any_key", ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringFromMap(tt.input, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMediaFile_Struct_Validation(t *testing.T) {
	now := time.Now()
	mediaFile := &MediaFile{
		ID:         "507f1f77bcf86cd799439011",
		Filename:   "test.jpg",
		Size:       1024,
		FileType:   common.MediaFileTypeImage,
		UploadedBy: "user123",
		UploadedAt: now,
	}
	
	assert.Equal(t, "507f1f77bcf86cd799439011", mediaFile.ID)
	assert.Equal(t, "test.jpg", mediaFile.Filename)
	assert.Equal(t, int64(1024), mediaFile.Size)
	assert.Equal(t, common.MediaFileTypeImage, mediaFile.FileType)
	assert.Equal(t, "user123", mediaFile.UploadedBy)
	assert.Equal(t, now, mediaFile.UploadedAt)
}

// Helper function to get environment variables with defaults
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

