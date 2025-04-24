package model

import (
	"gorm.io/gorm"
)

type UserSession struct {
	ID     int64  `gorm:"id" json:"id"`
	UserId int64  `gorm:"user_id" json:"user_id"`
	ToId   int64  `gorm:"to_id" json:"to_id"`
	Kind   string `gorm:"kind" json:"kind"`
	Seq    int64  `gorm:"seq" json:"seq"`
	gorm.Model
}

func (us UserSession) TableName() string {
	return "user_session"
}
