package model

import "gorm.io/gorm"

type GroupMember struct {
	ID        int64 `gorm:"id" json:"id"`
	GroupId   int64 `gorm:"group_id" json:"group_id"`
	UserId    int64 `gorm:"user_id" json:"user_id"`
	SessionId int64 `gorm:"session_id" json:"session_id"`
	gorm.Model
}

func (gm GroupMember) TableName() string {
	return "group_member"
}
