package config

import (
	"fmt"
	//     "log"
	//     "os"
	//     "strconv"
	//     "strings"
)

type Config struct {
	Server ServerConfig `json:"server"`

	// Database Configuration
	Database DatabaseConfig `json:"database"`

	// Firebase Configuration
	Firebase FirebaseConfig `json:"firebase"`

	// Notification Configuration
	Notification NotificationConfig `json:"notification"`

	// Email Configuration (optional)
	Email EmailConfig `json:"email"`

	// Logging Configuration
	Logging LoggingConfig `json:"logging"`
}

// ServerConfig contains server-related configuration
type ServerConfig struct {
	Port         string `json:"port"`
	Host         string `json:"host"`
	ReadTimeout  int    `json:"read_timeout"`
	WriteTimeout int    `json:"write_timeout"`
	Environment  string `json:"environment"` // development, staging, production
}

// DatabaseConfig contains database connection configuration
type DatabaseConfig struct {
	Host         string `json:"host"`
	Port         string `json:"port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	DatabaseName string `json:"database_name"`
	SSLMode      string `json:"ssl_mode"`
	MaxOpenConns int    `json:"max_open_conns"`
	MaxIdleConns int    `json:"max_idle_conns"`

	// Connection string will be built from above values
	DSN string `json:"-"`
}

// FirebaseConfig contains Firebase Cloud Messaging configuration
type FirebaseConfig struct {
	ProjectID           string `json:"project_id"`
	CredentialsFilePath string `json:"credentials_file_path"`
	Enabled             bool   `json:"enabled"`
}

// NotificationConfig contains notification system configuration
type NotificationConfig struct {
	Workers                int  `json:"workers"`                  // Number of worker goroutines
	ChannelBufferSize      int  `json:"channel_buffer_size"`      // Channel buffer size
	ScheduledCheckInterval int  `json:"scheduled_check_interval"` // Minutes
	MaxRetries             int  `json:"max_retries"`              // Max retry attempts
	RetryDelay             int  `json:"retry_delay"`              // Seconds
	Enabled                bool `json:"enabled"`
}

// EmailConfig contains email service configuration (optional)
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

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level      string `json:"level"`       // debug, info, warn, error
	Format     string `json:"format"`      // json, text
	OutputPath string `json:"output_path"` // stdout, stderr, or file path
}

func (cfg *Config) DSN() string {
	if cfg.Database.Host == "" {
		cfg.Database.Host = "localhost"
	}
	if cfg.Database.Port == "" {
		cfg.Database.Port = "3306"
	}

	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DatabaseName,
	)
}
