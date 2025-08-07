package dbmysql

import (
    "time"
    "gorm.io/gorm"
)

type User struct {
    UserID          uint64         `gorm:"primaryKey;column:user_id;autoIncrement" json:"user_id"`
    Handle          string         `gorm:"column:handle;uniqueIndex;size:50;not null" json:"handle"`
    PasswordHash    string         `gorm:"column:password_hash;size:255;not null" json:"-"`
    ProfileDetails  string         `gorm:"column:profile_details;type:text" json:"profile_details"`
    Email           string         `gorm:"column:email;size:255" json:"email"`
    Phone           string         `gorm:"column:phone;size:20" json:"phone"`
    Status          string         `gorm:"column:status;type:enum('active','banned','deleted');default:'active'" json:"status"`
    CreatedAt       time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt       time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
    DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

}
