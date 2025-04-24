package repository

import (
	"context"
	"go-im/internal/pkg/db"
	"go-im/internal/pkg/mtrace"
	"go-im/internal/user/model"
	"strings"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

const (
	FriendApplyStatusPending = iota + 1
	FriendApplyStatusAgree
	FriendApplyStatusReject
)

type FriendApplyRepository struct {
	db *db.DB
}

func NewFriendApplyRepository(db *db.DB) *FriendApplyRepository {
	return &FriendApplyRepository{db}
}

func (f *FriendApplyRepository) Insert(ctx context.Context, data *model.FriendApply) (int64, error) {
	err := f.db.Wrap(ctx, func() *gorm.DB {
		return f.db.Create(&data)
	})
	if err != nil {
		return 0, errors.Wrap(err, "Insert")
	}
	return data.ID, nil
}

func (f *FriendApplyRepository) UpdateFriendApply(ctx context.Context, id int64, status int) error {
	err := f.db.Wrap(ctx, func() *gorm.DB {
		return f.db.Model(&model.FriendApply{}).Where("id=?", id).Update("status=?", status)
	})
	if err != nil {
		return errors.Wrap(err, "UpdateFriendApply")
	}
	return nil
}

func (f *FriendApplyRepository) ListFriendApply(ctx context.Context, userId int64) ([]*model.FriendApply, error) {
	var friends []*model.FriendApply
	err := f.db.Wrap(ctx, func() *gorm.DB {
		return f.db.Where("(user_id=? or friend_id=?) and status=1").Order("created_at desc").Find(friends)
	})
	if err != nil {
		return nil, errors.Wrap(err, "ListFriendApply")
	}
	return friends, nil
}

func (f *FriendApplyRepository) GetFriendApply(ctx context.Context, id int64, userId int64) (*model.FriendApply, error) {
	var friend *model.FriendApply
	err := f.db.Wrap(ctx, func() *gorm.DB {
		return f.db.First(friend, "user_id=? and friend_id=?", id, userId)
	})
	if err != nil {
		return nil, errors.Wrap(err, "GetFriendApply")
	}
	return friend, nil
}

func (f *FriendApplyRepository) AgreeAndAddFriend(ctx context.Context, applyId int64, userId int64, friendId int64) error {
	err := f.db.Transaction(func(tx *gorm.DB) error {
		_, span := mtrace.StartSpan(ctx, "gorm", trace.WithSpanKind(trace.SpanKindInternal))
		defer mtrace.EndSpan(span)
		sql := make([]string, 0)
		defer func() {
			span.SetAttributes(mtrace.SQLKey.String(strings.Join(sql, "; ")))
		}()
		stmt := tx.Model(&model.FriendApply{}).Where("id=?", applyId).Update("status", FriendApplyStatusAgree)
		sql = append(sql, stmt.Statement.SQL.String())
		if stmt.Error != nil {
			return stmt.Error
		}
		batch := []*model.Friends{
			{
				UserId:   userId,
				FriendId: friendId,
			},
			{
				UserId:   friendId,
				FriendId: userId,
			},
		}
		stmt = tx.Create(batch)
		sql = append(sql, stmt.Statement.SQL.String())
		if stmt.Error != nil {
			return stmt.Error
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "AgreeAndAddFriend")
	}
	return nil
}

func (f *FriendApplyRepository) FindOne(ctx context.Context, id int64) (*model.FriendApply, error) {
	var apply *model.FriendApply
	err := f.db.Wrap(ctx, func() *gorm.DB {
		return f.db.First(&apply, "id=?")
	})
	if err != nil {
		return nil, errors.Wrap(err, "FindOne")
	}
	return apply, nil
}
