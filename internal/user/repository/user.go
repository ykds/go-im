package repository

import (
	"context"
	"go-im/internal/pkg/db"
	"go-im/internal/user/model"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *db.DB
}

func NewUserRepository(db *db.DB) *UserRepository {
	return &UserRepository{db}
}

func (u *UserRepository) Delete(ctx context.Context, id int64) error {
	err := u.db.Wrap(ctx, func() *gorm.DB {
		return u.db.Delete(&model.Users{}, "id=?", id)
	})
	if err != nil {
		return errors.Wrap(err, "Delete")
	}
	return nil
}

func (u *UserRepository) FindOne(ctx context.Context, id int64) (*model.Users, error) {
	var user *model.Users
	err := u.db.Wrap(ctx, func() *gorm.DB {
		return u.db.First(&user, "id=?", id)
	})
	if err != nil {
		return nil, errors.Wrap(err, "FindOne")
	}
	return user, nil
}

func (u *UserRepository) FindOneByPhone(ctx context.Context, phone string) (*model.Users, error) {
	var user *model.Users
	err := u.db.Wrap(ctx, func() *gorm.DB {
		return u.db.First(&user, "phone=?", phone)
	})
	if err != nil {
		return nil, errors.Wrap(err, "FindOneByPhone")
	}
	return user, nil
}

func (u *UserRepository) Insert(ctx context.Context, data *model.Users) (int64, error) {
	err := u.db.Wrap(ctx, func() *gorm.DB {
		return u.db.Create(&data)
	})
	if err != nil {
		return 0, errors.Wrap(err, "Insert")
	}
	return data.ID, nil
}

func (u *UserRepository) Update(ctx context.Context, newData *model.Users) error {
	err := u.db.Wrap(ctx, func() *gorm.DB {
		return u.db.Updates(&newData)
	})
	if err != nil {
		return errors.Wrap(err, "Update")
	}
	return nil
}
