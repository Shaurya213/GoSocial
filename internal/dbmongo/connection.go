// Package dbmongo is something
package dbmongo

import (
	"GoSocial/internal/config"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoClient struct {
	Client   *mongo.Client
	Database *mongo.Database
	GridFS   *gridfs.Bucket
}

func NewMongoConnection(c *config.Config) (*MongoClient, error) {
	uri := c.GetMongoURI()
	clientOptions := options.Client().ApplyURI(uri)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	database := client.Database(c.MongoDB.Database)
	bucket, err := gridfs.NewBucket(database, options.GridFSBucket().SetName("media_files"))
	if err != nil {
		return nil, fmt.Errorf("failed to create GridFSBucket: %w", err)
	}

	return &MongoClient{
		Client:   client,
		Database: database,
		GridFS:   bucket,
	}, nil
}

func (mc *MongoClient) Close(ctx context.Context) error {
	return mc.Client.Disconnect(ctx)
}
