package repository

import (
	"context"
	"go-im/internal/message/model"
	"go-im/internal/pkg/db"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type MessageRepository struct {
	db *db.DB
}

func NewMessageRepository(db *db.DB) *MessageRepository {
	return &MessageRepository{db}
}

func (m *MessageRepository) Insert(ctx context.Context, data *model.Message) (int64, error) {
	err := m.db.Wrap(ctx, "Insert", func(tx *gorm.DB) *gorm.DB {
		return m.db.Create(&data)
	})
	if err != nil {
		return 0, errors.Wrap(err, "Insert")
	}
	return data.ID, nil
}

func (m *MessageRepository) ListUnRead(ctx context.Context, toId int64, fromId int64, seq int64) ([]*model.Message, error) {
	var resp []*model.Message
	err := m.db.Wrap(ctx, "ListUnRead", func(tx *gorm.DB) *gorm.DB {
		return m.db.Find(&resp, "kind='single' AND from_id=? AND to_id=? AND seq >= ?", fromId, toId, seq)
	})
	if err != nil {
		return nil, errors.Wrap(err, "ListUnRead")
	}
	return resp, nil
}

func (m *MessageRepository) ListGroupUnRead(ctx context.Context, toId int64, seq int64) ([]*model.Message, error) {
	var resp []*model.Message
	err := m.db.Wrap(ctx, "ListGroupUnRead", func(tx *gorm.DB) *gorm.DB {
		return m.db.Find(&resp, "kind='group' AND to_id=? AND seq>=?", toId, seq)
	})
	if err != nil {
		return nil, errors.Wrap(err, "ListUnRead")
	}
	return resp, nil
}
