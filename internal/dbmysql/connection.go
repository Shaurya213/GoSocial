package dbmysql

import (
	"fmt"
	"gosocial/internal/config"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

/*
var db *gorm.DB

func SetDB(database *gorm.DB) {
	db = database
}

func GetDB() *gorm.DB {
	return db
}
*/

// NewMySQL returns a GORM DB instance connected to MySQL
func NewMySQL(cnf *config.Config) (*gorm.DB, error) {
	dsn := cnf.DSN()
	if dsn == "" {
		return nil, fmt.Errorf("MYSQL_DSN is not set")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:      logger.Default.LogMode(logger.Info),
		PrepareStmt: true,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot connect to MySQL: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("sql.DB error: %w", err)
	}
	sqlDB.SetMaxOpenConns(cnf.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cnf.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if err := db.AutoMigrate(&Message{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Println("✅ Connected to MySQL successfully")

	return db, nil
}
