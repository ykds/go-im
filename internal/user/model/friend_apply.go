package model

import "gorm.io/gorm"

type FriendApply struct {
	ID       int64  `gorm:"id" json:"id"`
	UserId   int64 `gorm:"user_id" json:"user_id"`
	FriendId int64 `gorm:"friend_id" json:"friend_id"`
	Status   int64  `gorm:"status" json:"status"`
	gorm.Model
}

func (u FriendApply) TableName() string {
	return "friend_apply"
}
