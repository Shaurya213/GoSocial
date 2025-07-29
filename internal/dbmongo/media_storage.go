package dbmongo

import (
	"bytes"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io"
)

var MediaBaseURL = "http://localhost:8080/media/"

func GetMediaURL(fileName string) string {
	return fmt.Sprintf("%s%s", MediaBaseURL, fileName)
}

type GridFSClient struct {
	bucket *gridfs.Bucket
}

func NewGridFSClient(db *mongo.Database) (*GridFSClient, error) {
	bucket, err := gridfs.NewBucket(db, options.GridFSBucket().SetName("media"))
	if err != nil {
		return nil, err
	}
	return &GridFSClient{bucket: bucket}, nil
}

func (c *GridFSClient) UploadFile(ctx context.Context, filename string, data []byte) (primitive.ObjectID, error) {
	uploadStream, err := c.bucket.OpenUploadStream(filename)
	if err != nil {
		return primitive.NilObjectID, err
	}
	defer uploadStream.Close()

	_, err = uploadStream.Write(data)
	if err != nil {
		return primitive.NilObjectID, err
	}

	return uploadStream.FileID.(primitive.ObjectID), nil
}

func (c *GridFSClient) DownloadFile(ctx context.Context, fileID string) ([]byte, error) {
	var buf bytes.Buffer
	dStream, err := c.bucket.OpenDownloadStreamByName(fileID)
	if err != nil {
		return nil, err
	}
	defer dStream.Close()

	_, err = io.Copy(&buf, dStream)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *GridFSClient) DeleteFile(ctx context.Context, fileIDStr string) error {
	objectID, err := primitive.ObjectIDFromHex(fileIDStr)
	if err != nil {
		return err
	}
	return c.bucket.Delete(objectID)
}

func (c *GridFSClient) GetFileByID(ctx context.Context, fileIDStr string) ([]byte, error) {
	// Step 1: Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(fileIDStr)
	if err != nil {
		return nil, err
	}

	// Step 2: Open download stream using ObjectID
	dStream, err := c.bucket.OpenDownloadStream(objectID)
	if err != nil {
		return nil, err
	}
	defer dStream.Close()

	// Step 3: Read file data into memory
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, dStream); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
