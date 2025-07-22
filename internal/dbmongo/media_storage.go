package dbmongo

import (
    "context"
    "fmt"
    "io"
    "time"

    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo/gridfs"
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
    ID       string
    Filename string
    Size     int64
    UploadedAt time.Time
}

// UploadMedia stores a media file in GridFS
func (ms *MediaStorage) UploadMedia(ctx context.Context, filename string, content io.Reader) (*MediaFile, error) {
    uploadStream, err := ms.gridFS.OpenUploadStream(filename)
    if err != nil {
        return nil, fmt.Errorf("failed to open upload stream: %w", err)
    }
    defer uploadStream.Close()

    size, err := io.Copy(uploadStream, content)
    if err != nil {
        return nil, fmt.Errorf("failed to upload file: %w", err)
    }

    return &MediaFile{
        ID:         uploadStream.FileID.(primitive.ObjectID).Hex(),
        Filename:   filename,
        Size:       size,
        UploadedAt: time.Now(),
    }, nil
}

// DownloadMedia retrieves a media file from GridFS
func (ms *MediaStorage) DownloadMedia(ctx context.Context, fileID string) (io.Reader, error) {
    objectID, err := primitive.ObjectIDFromHex(fileID)
    if err != nil {
        return nil, fmt.Errorf("invalid file ID: %w", err)
    }

    downloadStream, err := ms.gridFS.OpenDownloadStream(objectID)
    if err != nil {
        return nil, fmt.Errorf("failed to open download stream: %w", err)
    }

    return downloadStream, nil
}

// DeleteMedia removes a media file from GridFS
func (ms *MediaStorage) DeleteMedia(ctx context.Context, fileID string) error {
    objectID, err := primitive.ObjectIDFromHex(fileID)
    if err != nil {
        return fmt.Errorf("invalid file ID: %w", err)
    }

    return ms.gridFS.Delete(objectID)
}

