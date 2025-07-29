package dbmysql

import (
    "time"
    "gorm.io/gorm"
)

type Friend struct {
    ID               uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
    UserID           uint64     `gorm:"column:user_id;not null;index:idx_user_friend,unique" json:"user_id"`
    FriendUserID     uint64     `gorm:"column:friend_user_id;not null;index:idx_user_friend,unique" json:"friend_user_id"`
    Status           string     `gorm:"column:status;type:enum('pending','accepted','blocked');default:'pending'" json:"status"`
    RequestedAt      time.Time  `gorm:"column:requested_at;autoCreateTime" json:"requested_at"`
    AcceptedAt       *time.Time `gorm:"column:accepted_at" json:"accepted_at"`
    UpdatedAt        time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
    DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`

    // User       User `gorm:"foreignKey:UserID;references:UserID" json:"user"`
    // FriendUser User `gorm:"foreignKey:FriendUserID;references:UserID" json:"friend_user"`
    
    User       *User `gorm:"-" json:"user,omitempty"`
    FriendUser *User `gorm:"-" json:"friend_user,omitempty"`
}

