package dbmysql

import "gorm.io/gorm"

var db *gorm.DB

func SetDB(database *gorm.DB) {
	db = database
}

func GetDB() *gorm.DB {
	return db
}
