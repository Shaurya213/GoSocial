package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_DefaultBehavior(t *testing.T) {
	// Clean environment for testing defaults
	clearTestEnvVars()
	defer clearTestEnvVars()

	// Create a test .env file (since LoadConfig tries to load it)
	createTestEnvFile(t)
	defer removeTestEnvFile()

	config := LoadConfig()
	
	// Test that config is properly loaded and structured
	require.NotNil(t, config)
	require.NotNil(t, config.Database)
	require.NotNil(t, config.Server)
	require.NotNil(t, config.MongoDB)
	require.NotNil(t, config.Firebase)
	require.NotNil(t, config.Notification)
	require.NotNil(t, config.Email)
	require.NotNil(t, config.Logging)
	
	// Test database defaults (your Database struct, not MySQL)
	assert.Equal(t, "localhost", config.Database.Host)
	assert.Equal(t, "3306", config.Database.Port)
	assert.Equal(t, "gosocial", config.Database.Username)
	assert.Equal(t, "gosocial123", config.Database.Password)
	assert.Equal(t, "gosocial", config.Database.DatabaseName)
	assert.Equal(t, 25, config.Database.MaxOpenConns)
	assert.Equal(t, 5, config.Database.MaxIdleConns)

	// Test MongoDB defaults
	assert.Equal(t, "localhost", config.MongoDB.Host)
	assert.Equal(t, "27017", config.MongoDB.Port)
	assert.Equal(t, "admin", config.MongoDB.Username)
	assert.Equal(t, "admin123", config.MongoDB.Password)
	assert.Equal(t, "gosocial", config.MongoDB.Database)

	// Test server defaults
	assert.Equal(t, "7003", config.Server.ChatServicePort)
	assert.Equal(t, "7001", config.Server.UserServicePort)
	assert.Equal(t, "7002", config.Server.FeedServicePort)
	assert.Equal(t, "7004", config.Server.NotifServicePort)
	assert.Equal(t, "8080", config.Server.MediaServicePort)

	// Test notification defaults (hardcoded in your LoadConfig)
	assert.Equal(t, 5, config.Notification.Workers)
	assert.Equal(t, 1000, config.Notification.ChannelBufferSize)
	assert.Equal(t, 1, config.Notification.ScheduledCheckInterval)
	assert.Equal(t, 3, config.Notification.MaxRetries)
	assert.Equal(t, 5, config.Notification.RetryDelay)
	assert.True(t, config.Notification.Enabled)

	// Test that MEDIA_BASE_URL was set dynamically
	assert.NotEmpty(t, config.Server.MediaBaseURL)
	assert.Contains(t, config.Server.MediaBaseURL, "/media")
}

func TestLoadConfig_WithEnvironmentOverrides(t *testing.T) {
	// Set test environment variables
	testEnvVars := map[string]string{
		"MYSQL_HOST":         "test-db-host",
		"MYSQL_PORT":         "3307",
		"MYSQL_USERNAME":     "test-user",
		"MYSQL_PASSWORD":     "test-pass",
		"MYSQL_DATABASE":     "test-db",
		"MONGO_HOST":         "test-mongo",
		"MONGO_PORT":         "27018",
		"MONGO_USERNAME":     "mongo-user",
		"MONGO_PASSWORD":     "mongo-pass",
		"MONGO_DATABASE":     "mongo-test",
		"CHAT_SERVICE_PORT":  "7010",
		"USER_SERVICE_PORT":  "7011",
		"FEED_SERVICE_PORT":  "7012",
		"NOTIF_SERVICE_PORT": "7013",
		"MEDIA_SERVER_PORT":  "8090",
		"FIREBASE_PROJECT_ID": "test-firebase",
		"FIREBASE_ENABLED":   "true",
		"SMTP_HOST":          "smtp.test.com",
		"EMAIL_ENABLED":      "true",
		"LOG_LEVEL":          "debug",
	}

	for key, value := range testEnvVars {
		os.Setenv(key, value)
	}
	
	// Create test .env file
	createTestEnvFile(t)
	defer removeTestEnvFile()
	
	defer func() {
		for key := range testEnvVars {
			os.Unsetenv(key)
		}
		clearTestEnvVars()
	}()
	
	config := LoadConfig()
	
	// Verify environment variables were loaded
	assert.Equal(t, "test-db-host", config.Database.Host)
	assert.Equal(t, "3307", config.Database.Port)
	assert.Equal(t, "test-user", config.Database.Username)
	assert.Equal(t, "test-mongo", config.MongoDB.Host)
	assert.Equal(t, "27018", config.MongoDB.Port)
	assert.Equal(t, "7010", config.Server.ChatServicePort)
	assert.Equal(t, "test-firebase", config.Firebase.ProjectID)
	assert.True(t, config.Firebase.Enabled)
	assert.Equal(t, "smtp.test.com", config.Email.SMTPHost)
	assert.True(t, config.Email.Enabled)
	assert.Equal(t, "debug", config.Logging.Level)
}

func TestDSN_Generation(t *testing.T) {
	config := &Config{
		Database: DatabaseConfig{
			Host:         "test-host",
			Port:         "3307",
			Username:     "testuser",
			Password:     "testpass",
			DatabaseName: "testdb",
		},
	}
	
	dsn := config.DSN()
	expected := "testuser:testpass@tcp(test-host:3307)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
	assert.Equal(t, expected, dsn)
}

func TestDSN_WithEmptyHostPort(t *testing.T) {
	config := &Config{
		Database: DatabaseConfig{
			Username:     "testuser",
			Password:     "testpass",
			DatabaseName: "testdb",
			// Host and Port are empty - should default
		},
	}
	
	dsn := config.DSN()
	// Should default to localhost:3306
	expected := "testuser:testpass@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
	assert.Equal(t, expected, dsn)
}

func TestGetMongoURI_WithAuth(t *testing.T) {
	config := &Config{
		MongoDB: MongoDBConfig{
			Host:     "mongo-host",
			Port:     "27017",
			Username: "mongouser",
			Password: "mongopass",
			Database: "mongodb",
		},
	}
	
	uri := config.GetMongoURI()
	expected := "mongodb://mongouser:mongopass@mongo-host:27017/mongodb?authSource=admin"
	assert.Equal(t, expected, uri)
}

func TestGetMongoURI_WithoutAuth(t *testing.T) {
	config := &Config{
		MongoDB: MongoDBConfig{
			Host:     "mongo-host",
			Port:     "27017",
			Username: "",
			Password: "",
			Database: "mongodb",
		},
	}
	
	uri := config.GetMongoURI()
	expected := "mongodb://mongo-host:27017/mongodb"
	assert.Equal(t, expected, uri)
}

func TestGetEnv_HelperFunction(t *testing.T) {
	// Test with existing env var
	os.Setenv("TEST_KEY", "test_value")
	defer os.Unsetenv("TEST_KEY")
	
	result := getEnv("TEST_KEY", "default_value")
	assert.Equal(t, "test_value", result)
	
	// Test with non-existent env var
	result = getEnv("NON_EXISTENT_KEY", "default_value")
	assert.Equal(t, "default_value", result)
	
	// Test with empty env var
	os.Setenv("EMPTY_KEY", "")
	defer os.Unsetenv("EMPTY_KEY")
	
	result = getEnv("EMPTY_KEY", "default_value")
	assert.Equal(t, "default_value", result)
}

func TestGetEnvAsInt_HelperFunction(t *testing.T) {
	// Test with valid integer
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")
	
	result := getEnvAsInt("TEST_INT", 10)
	assert.Equal(t, 42, result)
	
	// Test with invalid integer
	os.Setenv("INVALID_INT", "not-a-number")
	defer os.Unsetenv("INVALID_INT")
	
	result = getEnvAsInt("INVALID_INT", 10)
	assert.Equal(t, 10, result)
	
	// Test with non-existent key
	result = getEnvAsInt("NON_EXISTENT_INT", 100)
	assert.Equal(t, 100, result)
}

func TestConfigStructure_AllFieldsPresent(t *testing.T) {
	createTestEnvFile(t)
	defer removeTestEnvFile()
	
	config := LoadConfig()
	
	// Test that all major config sections are present
	assert.NotNil(t, config.Server)
	assert.NotNil(t, config.Database)
	assert.NotNil(t, config.Firebase)
	assert.NotNil(t, config.Notification)
	assert.NotNil(t, config.Email)
	assert.NotNil(t, config.Logging)
	assert.NotNil(t, config.MongoDB)
}

// Test helper functions
func createTestEnvFile(t *testing.T) {
	content := `# Test .env file
MONGO_HOST=localhost
MYSQL_HOST=localhost
`
	err := os.WriteFile(".env", []byte(content), 0644)
	require.NoError(t, err)
}

func removeTestEnvFile() {
	os.Remove(".env")
}

func clearTestEnvVars() {
	envKeys := []string{
		"MYSQL_HOST", "MYSQL_PORT", "MYSQL_USERNAME", "MYSQL_PASSWORD", "MYSQL_DATABASE",
		"MONGO_HOST", "MONGO_PORT", "MONGO_USERNAME", "MONGO_PASSWORD", "MONGO_DATABASE",
		"CHAT_SERVICE_PORT", "USER_SERVICE_PORT", "FEED_SERVICE_PORT", "NOTIF_SERVICE_PORT", "MEDIA_SERVER_PORT",
		"FIREBASE_PROJECT_ID", "FIREBASE_CREDENTIALS_PATH", "FIREBASE_ENABLED",
		"SMTP_HOST", "SMTP_PORT", "SMTP_USERNAME", "SMTP_PASSWORD", "FROM_EMAIL", "FROM_NAME", "EMAIL_ENABLED",
		"LOG_LEVEL", "LOG_FORMAT", "LOG_OUTPUT", "MEDIA_BASE_URL",
	}
	
	for _, key := range envKeys {
		os.Unsetenv(key)
	}
}

