package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	//     "log"
	//     "os"
	//     "strconv"
	//     "strings"
)

type Config struct {
	Server ServerConfig `json:"server"`

	Database DatabaseConfig `json:"database"`

	Firebase FirebaseConfig `json:"firebase"`

	Notification NotificationConfig `json:"notification"`

	Email EmailConfig `json:"email"`

	Logging LoggingConfig `json:"logging"`
}

type ServerConfig struct {
	ChatServicePort  string
	UserServicePort  string
	FeedServicePort  string
	NotifServicePort string
	MediaServicePort string
	MediaBaseURL     string
}

type DatabaseConfig struct {
	Host         string `json:"host"`
	Port         string `json:"port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	DatabaseName string `json:"database_name"`
	SSLMode      string `json:"ssl_mode"`
	MaxOpenConns int    `json:"max_open_conns"`
	MaxIdleConns int    `json:"max_idle_conns"`

	DSN string `json:"-"`
}

type FirebaseConfig struct { // FirebaseConfig contains Firebase Cloud Messaging configuration
	ProjectID           string `json:"project_id"`
	CredentialsFilePath string `json:"credentials_file_path"`
	Enabled             bool   `json:"enabled"`
}

type NotificationConfig struct {
	Workers                int  `json:"workers"`                  // Number of worker goroutines
	ChannelBufferSize      int  `json:"channel_buffer_size"`      // Channel buffer size
	ScheduledCheckInterval int  `json:"scheduled_check_interval"` // Minutes
	MaxRetries             int  `json:"max_retries"`              // Max retry attempts
	RetryDelay             int  `json:"retry_delay"`              // Seconds
	Enabled                bool `json:"enabled"`
}

type EmailConfig struct {
	SMTPHost  string `json:"smtp_host"`
	SMTPPort  int    `json:"smtp_port"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	FromEmail string `json:"from_email"`
	FromName  string `json:"from_name"`
	Enabled   bool   `json:"enabled"`
	UseTLS    bool   `json:"use_tls"`
}

type LoggingConfig struct {
	Level      string `json:"level"`  // e.g., "info", "debug", "warn", "error"
	Format     string `json:"format"` // e.g., "json", "text"
	OutputPath string `json:"output"` // e.g., "stdout", "logfile.log"
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	return &Config{
		Database: DatabaseConfig{
			Host:         getEnv("DB_HOST", "192.168.63.59"),
			Port:         getEnv("DB_PORT", "3306"),
			Username:     getEnv("DB_USER", "gosocial_user"),
			Password:     getEnv("DB_PASSWORD", "G0Social@123"),
			DatabaseName: getEnv("DB_NAME", "gosocial_db"),
			MaxOpenConns: 25,
			MaxIdleConns: 5,
		},
		Firebase: FirebaseConfig{
			ProjectID:           getEnv("FIREBASE_PROJECT_ID", ""),
			CredentialsFilePath: getEnv("FIREBASE_CREDENTIALS_PATH", ""),
			Enabled:             getEnv("FIREBASE_ENABLED", "false") == "true",
		},
		Notification: NotificationConfig{
			Workers:                5,
			ChannelBufferSize:      1000,
			ScheduledCheckInterval: 1,
			MaxRetries:             3,
			RetryDelay:             5,
			Enabled:                true,
		},
		Email: EmailConfig{
			SMTPHost:  getEnv("SMTP_HOST", ""),
			SMTPPort:  getEnvAsInt("SMTP_PORT", 587),
			Username:  getEnv("SMTP_USERNAME", ""),
			Password:  getEnv("SMTP_PASSWORD", ""),
			FromEmail: getEnv("FROM_EMAIL", ""),
			FromName:  getEnv("FROM_NAME", "GoSocial"),
			Enabled:   getEnv("EMAIL_ENABLED", "false") == "true",
			UseTLS:    true,
		},
		Logging: LoggingConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "text"),
			OutputPath: getEnv("LOG_OUTPUT", "stdout"),
		},
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return fallback
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return fallback
	}
	return val
}
