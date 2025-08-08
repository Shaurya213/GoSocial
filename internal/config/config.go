package config

import (
	"fmt"

	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server ServerConfig `json:"server"`
	MongoDB MongoDBConfig
	Database DatabaseConfig `json:"database"`
	Firebase FirebaseConfig `json:"firebase"`
	Notification NotificationConfig `json:"notification"`
	Email EmailConfig `json:"email"`
	Logging LoggingConfig `json:"logging"`
	JWT     JWTConfig
}

type MongoDBConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
}

type JWTConfig struct {
	Secret string
}


type ServerConfig struct {
	ChatServicePort  string
	UserServicePort  string
	FeedServicePort  string
	NotifServicePort string
	MediaServicePort string
	MediaBaseURL	 string
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
)

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil{
		log.Fatalf(".env is not laoding: %v", err)
	}

	cmd := exec.Command("bash", "-c", "curl ifconfig.me")
	out, _ := cmd.Output()
	ip:= string(out)
	//ip = "localhost"

	//Setting new envs
	//os.Setenv("MONGO_HOST", ip)
	//os.Setenv("MYSQL_HOST", ip)
	os.Setenv("MEDIA_BASE_URL", fmt.Sprintf("http://%s:%s/media", ip, os.Getenv("MEDIA_SERVER_PORT")))

	return &Config{
		MongoDB: MongoDBConfig{
			Host:     getEnv("MONGO_HOST", "localhost"),
			Port:     getEnv("MONGO_PORT", "27017"),
			Username: getEnv("MONGO_USERNAME", "admin"),
			Password: getEnv("MONGO_PASSWORD", "admin123"),
			Database: getEnv("MONGO_DATABASE", "gosocial"),
		},
		MySQL: MySQLConfig{
			Host:     getEnv("MYSQL_HOST", "localhost"),
			Port:     getEnv("MYSQL_PORT", "3306"),
			Username: getEnv("MYSQL_USERNAME", "gosocial"),
			Password: getEnv("MYSQL_PASSWORD", "gosocial123"),
			Database: getEnv("MYSQL_DATABASE", "gosocial"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "default-secret-change-in-production"),
		},
		Server: ServerConfig{
			ChatServicePort:  getEnv("CHAT_SERVICE_PORT", "7003"),
			UserServicePort:  getEnv("USER_SERVICE_PORT", "7001"),
			FeedServicePort:  getEnv("FEED_SERVICE_PORT", "7002"),
			NotifServicePort: getEnv("NOTIF_SERVICE_PORT", "7004"),
			MediaServicePort: getEnv("MEDIA_SERVER_PORT", "8080"),
			MediaBaseURL: getEnv("MEDIA_BASE_URL", "http://localhost:8080/media"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		fmt.Println(value)
		return value
	}
	return defaultValue
}

func (c *Config) GetMongoURI() string {
	if c.MongoDB.Username != "" && c.MongoDB.Password != "" {
		return fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=admin",
		c.MongoDB.Username, c.MongoDB.Password,
		c.MongoDB.Host, c.MongoDB.Port, c.MongoDB.Database)
	}
	return fmt.Sprintf("mongodb://%s:%s/%s",
	c.MongoDB.Host, c.MongoDB.Port, c.MongoDB.Database)
}

func (c *Config) GetMySQLDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
	c.MySQL.Username, c.MySQL.Password,
	c.MySQL.Host, c.MySQL.Port, c.MySQL.Database)

}
