package repository

import (
	"context"
	"go-im/internal/message/model"
	"go-im/internal/pkg/db"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type UserGroupRepository struct {
	db *db.DB
}

func NewUserGroupRepository(db *db.DB) *UserGroupRepository {
	return &UserGroupRepository{db}
}

func (u *UserGroupRepository) ListUserGroup(ctx context.Context, userId int64) ([]*model.UserGroup, error) {
	var result []*model.UserGroup
	err := u.db.Wrap(ctx, func() *gorm.DB {
		return u.db.Find(&result, "user_id=?", userId)
	})
	if err != nil {
		return nil, errors.Wrap(err, "ListUserGroup")
	}
	return result, nil
}
