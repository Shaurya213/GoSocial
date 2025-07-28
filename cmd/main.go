package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"gosocial/internal/config"
	"gosocial/internal/dbmysql"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	if os.Getenv("DB_HOST") == "" || os.Getenv("DB_USER") == "" {
		log.Println("Warning: Some database environment variables might be missing")
		log.Printf("DB_HOST: '%s', DB_USER: '%s'", os.Getenv("DB_HOST"), os.Getenv("DB_USER"))
	}

	log.Println("Connecting to database...")
	db, err := gorm.Open(mysql.Open(config.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	dbmysql.SetDB(db)
	log.Println("Database connection established successfully!")

	log.Println("Running database migration...")
	if err := db.AutoMigrate(&dbmysql.Notification{}); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Database migration completed successfully!")
	log.Println("Application initialized successfully!")

	// Optional: Test the database connection
	sqlDB, err := db.DB()
	if err == nil {
		if err := sqlDB.Ping(); err != nil {
			log.Printf("Warning: Database ping failed: %v", err)
		} else {
			log.Println("Database ping successful!")
		}
	}

	log.Println("Application is ready to serve requests...")
}
