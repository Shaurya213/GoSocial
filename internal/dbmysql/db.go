package dbmysql

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewMySQL returns a GORM DB instance connected to MySQL
func NewMySQL() (*gorm.DB, error) {
	dsn := os.Getenv("gosocial_user:GOS0cial@123@tcp(192.168.63.59:3306)/gosocial_db?parseTime=true") // e.g. "user:pass@tcp(localhost:3306)/gosocial?parseTime=true"
	if dsn == "" {
		return nil, fmt.Errorf("MYSQL_DSN is not set")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:      logger.Default.LogMode(logger.Info), // can change to Error/Warn for prod
		PrepareStmt: true,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot connect to MySQL: %w", err)
	}

	// Optional: Set connection pool config
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("sql.DB error: %w", err)
	}

	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	log.Println("✅ Connected to MySQL successfully")
	return db, nil
}
