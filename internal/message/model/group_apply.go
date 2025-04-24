package model

import (
	"gorm.io/gorm"
)

type GroupApply struct {
	ID      int64  `gorm:"id" json:"id"`
	GroupId int64  `gorm:"group_id" json:"group_id"`
	UserId  int64  `gorm:"user_id" json:"user_id"`
	Status  string `gorm:"status" json:"status"`
	Comment string `gorm:"comment" json:"comment"`
	gorm.Model
}

func (ga GroupApply) TableName() string {
	return "group_apply"
}