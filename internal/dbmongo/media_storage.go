package dbmongo

import (
	"context"
	"fmt"
	"io"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MediaStorage struct {
	gridFS  *gridfs.Bucket
	baseURL string
}

// Simple constructor
func NewMediaStorage(mongoClient *MongoClient, baseURL ...string) *MediaStorage {
	url := "http://localhost:8080/media" // Default URL
	if len(baseURL) > 0 {
		url = baseURL[0]
	}

	return &MediaStorage{
		gridFS:  mongoClient.GridFS,
		baseURL: url,
	}
}

// Simple file info struct - only what you need
type MediaFile struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	UploadedBy  string    `json:"uploaded_by"`
	URL         string    `json:"url"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

// Upload file with basic metadata (all you need for GoSocial)
func (ms *MediaStorage) UploadFile(ctx context.Context, filename, contentType, uploaderID string, content io.Reader) (*MediaFile, error) {
	// Basic metadata for GridFS
	metadata := bson.M{
		"content_type": contentType,
		"uploaded_by":  uploaderID,
		"uploaded_at":  time.Now(),
	}

	uploadOptions := options.GridFSUpload().SetMetadata(metadata)
	uploadStream, err := ms.gridFS.OpenUploadStream(filename, uploadOptions)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	defer uploadStream.Close()

	size, err := io.Copy(uploadStream, content)
	if err != nil {
		return nil, fmt.Errorf("file copy failed: %w", err)
	}

	fileID := uploadStream.FileID.(primitive.ObjectID).Hex()

	return &MediaFile{
		ID:          fileID,
		Filename:    filename,
		Size:        size,
		ContentType: contentType,
		UploadedBy:  uploaderID,
		URL:         fmt.Sprintf("%s/%s", ms.baseURL, fileID),
		UploadedAt:  time.Now(),
	}, nil
}

// Download file (simple version)
func (ms *MediaStorage) DownloadFile(ctx context.Context, fileID string) (io.Reader, *MediaFile, error) {
	objectID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file ID: %w", err)
	}

	downloadStream, err := ms.gridFS.OpenDownloadStream(objectID)
	if err != nil {
		return nil, nil, fmt.Errorf("download failed: %w", err)
	}

	// Get basic file info
	fileInfo := downloadStream.GetFile()

	// Simple metadata extraction
	var metadata bson.M
	if fileInfo.Metadata != nil {
		bson.Unmarshal(fileInfo.Metadata, &metadata)
	}

	mediaFile := &MediaFile{
		ID:          fileID,
		Filename:    fileInfo.Name,
		Size:        fileInfo.Length,
		URL:         fmt.Sprintf("%s/%s", ms.baseURL, fileID),
		UploadedAt:  fileInfo.UploadDate,
		ContentType: getStringFromMap(metadata, "content_type"),
		UploadedBy:  getStringFromMap(metadata, "uploaded_by"),
	}

	return downloadStream, mediaFile, nil
}

// Delete file
func (ms *MediaStorage) DeleteFile(ctx context.Context, fileID string) error {
	objectID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return fmt.Errorf("invalid file ID: %w", err)
	}
	return ms.gridFS.Delete(objectID)
}

// Simple helper function
func getStringFromMap(m bson.M, key string) string {
	if m == nil {
		return ""
	}
	if value, ok := m[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

