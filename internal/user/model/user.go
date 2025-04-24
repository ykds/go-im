package model

import (
	"gorm.io/gorm"
)

type Users struct {
	ID       int64  `gorm:"id" json:"id"`
	Username string `gorm:"username" json:"username"`
	Phone    string `gorm:"phone" json:"phone"`
	Gender   string `gorm:"gender" json:"gender"`
	Avatar   string `gorm:"avatar" json:"avatar"`
	Password string `gorm:"password" json:"password"`
	gorm.Model
}

func (u Users) TableName() string {
	return "users"
}
