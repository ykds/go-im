package repository

import (
	"context"
	"go-im/internal/pkg/db"
	"go-im/internal/user/model"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type FriendRepository struct {
	db *db.DB
}

func NewFriendRepository(db *db.DB) *FriendRepository {
	return &FriendRepository{db}
}

func (f *FriendRepository) Delete(ctx context.Context, id uint64) error {
	err := f.db.Wrap(ctx, "Delete", func(tx *gorm.DB) *gorm.DB {
		return tx.Delete(&model.Friends{}, "id=?", id)
	})
	if err != nil {
		return errors.Wrap(err, "Delete")
	}
	return nil
}

func (f *FriendRepository) FindOne(ctx context.Context, id uint64) (*model.Friends, error) {
	var friend *model.Friends
	err := f.db.Wrap(ctx, "FindOne", func(tx *gorm.DB) *gorm.DB {
		return tx.First(&friend, "id=?", id)
	})
	if err != nil {
		return nil, errors.Wrap(err, "FindOne")
	}
	return friend, nil
}

func (f *FriendRepository) Insert(ctx context.Context, data *model.Friends) (int64, error) {
	err := f.db.Wrap(ctx, "Insert", func(tx *gorm.DB) *gorm.DB {
		return tx.Create(&data)
	})
	if err != nil {
		return 0, errors.Wrap(err, "Insert")
	}
	return data.ID, nil
}

func (f *FriendRepository) UpdateFriendInfo(ctx context.Context, data *model.Friends) error {
	err := f.db.Wrap(ctx, "UpdateFriendInfo", func(tx *gorm.DB) *gorm.DB {
		return tx.Updates(&data)
	})
	if err != nil {
		return errors.Wrap(err, "UpdateFriendInfo")
	}
	return nil
}

func (f *FriendRepository) ListFriends(ctx context.Context, userId int64) ([]*model.Friends, error) {
	resp := make([]*model.Friends, 0)
	err := f.db.Wrap(ctx, "ListFriends", func(tx *gorm.DB) *gorm.DB {
		return tx.Find(&resp, "user_id=?", userId)
	})
	if err != nil {
		return nil, errors.Wrap(err, "ListFriends")
	}
	return resp, nil
}

func (f *FriendRepository) GetFriendById(ctx context.Context, userId int64, friendId int64) (*model.Friends, error) {
	var friend *model.Friends
	err := f.db.Wrap(ctx, "GetFriendById", func(tx *gorm.DB) *gorm.DB {
		return tx.First(&friend, "user_id = ? AND friend_id=?", userId, friendId)
	})
	if err != nil {
		return nil, errors.Wrap(err, "GetFriendById")
	}
	return friend, nil
}
