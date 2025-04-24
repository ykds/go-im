package model

import (
	"gorm.io/gorm"
)

type Group struct {
	ID      int64  `gorm:"id" json:"id"`
	GroupNo int64  `gorm:"group_no" json:"group_no"`
	Name    string `gorm:"name" json:"name"`
	OwnerId int64  `gorm:"owner_id" json:"owner_id"`
	Avatar  string `gorm:"avatar" json:"avatar"`
	gorm.Model
}

func (g Group) TableName() string {
	return "group"
}
