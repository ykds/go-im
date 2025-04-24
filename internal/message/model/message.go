package model

import "gorm.io/gorm"

type Message struct {
	ID      int64  `gorm:"id" json:"id"`
	FromId  int64  `gorm:"from_id" json:"from_id"`
	ToId    int64  `gorm:"to_id" json:"to_id"`
	Content string `gorm:"content" json:"content"`
	Seq     int64  `gorm:"seq" json:"seq"`
	Kind    string `gorm:"kind" json:"kind"`
	gorm.Model
}

func (m Message) TableName() string {
	return "message"
}
