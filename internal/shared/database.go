package shared

import (
	"fmt"
	"gosocial/internal/dbmysql"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DatabaseConfig holds the config values from .env
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

// Load DB config from environment variables
func GetDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Database: os.Getenv("DB_NAME"),
	}
}

// InitDatabase connects to DB and applies auto migrations
func InitDatabase(config DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=True&loc=Local",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.AutoMigrate(
		&dbmysql.User{},
		&dbmysql.Friend{},
		&dbmysql.Device{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database Connected & Migrated Successfully")
	return db, nil
}
