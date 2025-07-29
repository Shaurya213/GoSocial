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

	"gosocial/internal/common"
)

type MediaStorage struct {
	gridFS *gridfs.Bucket
}

func NewMediaStorage(mongoClient *MongoClient) *MediaStorage {
	return &MediaStorage{
		gridFS: mongoClient.GridFS,
	}
}

type MediaFile struct {
	ID         string               `json:"id"`          // GridFS ObjectID
	Filename   string               `json:"filename"`    // Original filename
	Size       int64                `json:"size"`        // File size in bytes
	FileType   common.MediaFileType `json:"file_type"`   // image or video (from PDF)
	UploadedBy string               `json:"uploaded_by"` // User ID who uploaded
	UploadedAt time.Time            `json:"uploaded_at"` // Upload timestamp
}

func (ms *MediaStorage) UploadFile(ctx context.Context, filename, mimeType, uploaderID string, content io.Reader) (*MediaFile, error) {
	// Detect file type based on MIME
	fileType := common.DetectFileType(mimeType)

	// Basic metadata for GridFS (matching PDF MediaRef schema)
	metadata := bson.M{
		"file_type":   fileType.String(), // "image" or "video"
		"mime_type":   mimeType,          // Full MIME type
		"uploaded_by": uploaderID,        // User who uploaded
		"uploaded_at": time.Now(),        // Upload timestamp
	}

	// Upload to GridFS
	opts := options.GridFSUpload().SetMetadata(metadata)
	stream, err := ms.gridFS.OpenUploadStream(filename, opts)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	defer stream.Close()

	// Copy file content
	size, err := io.Copy(stream, content)
	if err != nil {
		return nil, fmt.Errorf("file copy failed: %w", err)
	}

	// Return MediaFile info
	return &MediaFile{
		ID:         stream.FileID.(primitive.ObjectID).Hex(),
		Filename:   filename,
		Size:       size,
		FileType:   fileType,
		UploadedBy: uploaderID,
		UploadedAt: time.Now(),
	}, nil
}

func (ms *MediaStorage) DownloadFile(ctx context.Context, fileID string) (io.Reader, *MediaFile, error) {
	objectID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid file ID: %w", err)
	}

	// Open download stream
	stream, err := ms.gridFS.OpenDownloadStream(objectID)
	if err != nil {
		return nil, nil, fmt.Errorf("download failed: %w", err)
	}

	// Get file metadata
	fileInfo := stream.GetFile()
	var metadata bson.M
	if fileInfo.Metadata != nil {
		bson.Unmarshal(fileInfo.Metadata, &metadata)
	}

	// Build MediaFile info
	mediaFile := &MediaFile{
		ID:         fileID,
		Filename:   fileInfo.Name,
		Size:       fileInfo.Length,
		FileType:   common.MediaFileType(getStringFromMap(metadata, "file_type")),
		UploadedBy: getStringFromMap(metadata, "uploaded_by"),
		UploadedAt: fileInfo.UploadDate,
	}

	return stream, mediaFile, nil
}

func (ms *MediaStorage) DeleteFile(ctx context.Context, fileID string) error {
	objectID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return fmt.Errorf("invalid file ID: %w", err)
	}
	return ms.gridFS.Delete(objectID)
}

// Helper function for metadata extraction
func getStringFromMap(m bson.M, key string) string {
	if m == nil {
		return ""
	}
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
