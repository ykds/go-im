package model

import "gorm.io/gorm"

type Friends struct {
	ID       int64  `gorm:"id" json:"id"`
	UserId   int64  `gorm:"user_id" json:"user_id"`
	FriendId int64  `gorm:"friend_id" json:"friend_id"`
	Remark   string `gorm:"remark" json:"remark"`
	gorm.Model
}
