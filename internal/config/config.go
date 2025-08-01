package config

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoDB MongoDBConfig
	MySQL   MySQLConfig
	JWT     JWTConfig
	Server  ServerConfig
}

type MongoDBConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
}

type MySQLConfig struct {
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
