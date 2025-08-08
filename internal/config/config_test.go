package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Set test environment variables
	os.Setenv("MONGO_HOST", "test-mongo")
	os.Setenv("MONGO_PORT", "27018")
	os.Setenv("MYSQL_HOST", "test-mysql")
	os.Setenv("MYSQL_PORT", "3307")
	os.Setenv("JWT_SECRET", "test-secret")
	
	defer func() {
		// Cleanup
		os.Unsetenv("MONGO_HOST")
		os.Unsetenv("MONGO_PORT")
		os.Unsetenv("MYSQL_HOST")
		os.Unsetenv("MYSQL_PORT")
		os.Unsetenv("JWT_SECRET")
	}()

	config := LoadConfig()

	assert.Equal(t, "test-mongo", config.MongoDB.Host)
	assert.Equal(t, "27018", config.MongoDB.Port)
	assert.Equal(t, "test-mysql", config.MySQL.Host)
	assert.Equal(t, "3307", config.MySQL.Port)
	assert.Equal(t, "test-secret", config.JWT.Secret)
}

func TestConfig_GetMongoURI(t *testing.T) {
	config := &Config{
		MongoDB: MongoDBConfig{
			Host:     "localhost",
			Port:     "27017",
			Username: "admin",
			Password: "pass123",
			Database: "testdb",
		},
	}

	uri := config.GetMongoURI()
	expected := "mongodb://admin:pass123@localhost:27017/testdb?authSource=admin"
	assert.Equal(t, expected, uri)
}

func TestConfig_GetMySQLDSN(t *testing.T) {
	config := &Config{
		MySQL: MySQLConfig{
			Host:     "localhost",
			Port:     "3306",
			Username: "testuser",
			Password: "testpass",
			Database: "testdb",
		},
	}

	dsn := config.GetMySQLDSN()
	expected := "testuser:testpass@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
	assert.Equal(t, expected, dsn)
}

