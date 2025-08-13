package dbmongo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gosocial/internal/config"
)

func TestMongoClient_Structure(t *testing.T) {
	// Test struct initialization
	client := &MongoClient{}
	assert.NotNil(t, client)
}

func TestNewMongoConnection_ConfigValidation(t *testing.T) {
	// Test with valid config structure
	cfg := &config.Config{
		MongoDB: config.MongoDBConfig{
			Host:     "localhost",
			Port:     "27017",
			Username: "",
			Password: "",
			Database: "testdb",
		},
	}
	
	// Test URI generation
	uri := cfg.GetMongoURI()
	assert.Contains(t, uri, "mongodb://localhost:27017")
	assert.Contains(t, uri, "testdb")
}

func TestNewMongoConnection_WithAuth(t *testing.T) {
	cfg := &config.Config{
		MongoDB: config.MongoDBConfig{
			Host:     "localhost",
			Port:     "27017",
			Username: "admin",
			Password: "pass123",
			Database: "testdb",
		},
	}
	
	uri := cfg.GetMongoURI()
	expected := "mongodb://admin:pass123@localhost:27017/testdb?authSource=admin"
	assert.Equal(t, expected, uri)
}

func TestNewMongoConnection_WithoutAuth(t *testing.T) {
	cfg := &config.Config{
		MongoDB: config.MongoDBConfig{
			Host:     "localhost", 
			Port:     "27017",
			Username: "",
			Password: "",
			Database: "testdb",
		},
	}
	
	uri := cfg.GetMongoURI()
	expected := "mongodb://localhost:27017/testdb"
	assert.Equal(t, expected, uri)
}

