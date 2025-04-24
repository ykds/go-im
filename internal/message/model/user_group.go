package model

import "gorm.io/gorm"

type UserGroup struct {
	ID      int64 `gorm:"id" json:"id"`
	UserId  int64 `gorm:"user_id" json:"user_id"`
	GroupId int64 `gorm:"group_id" json:"group_id"`
	gorm.Model
}

func (ug UserGroup) TableName() string {
	return "user_group"
}
