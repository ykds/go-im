package repository

import (
	"context"
	"go-im/internal/message/model"
	"go-im/internal/pkg/db"
	"go-im/internal/pkg/mtrace"
	"strings"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"

	"gorm.io/gorm"
)

const (
	GroupApplyWaitStatus   = "wait"
	GroupApplyAccpedStatus = "accpeted"
	GroupApplyRejectStatus = "rejected"
)

type GroupApplyRepository struct {
	db *db.DB
}

func NewGroupApplyRepository(db *db.DB) *GroupApplyRepository {
	return &GroupApplyRepository{db}
}

func (g *GroupApplyRepository) FindOne(ctx context.Context, id int64) (*model.GroupApply, error) {
	var groupApply *model.GroupApply
	err := g.db.Wrap(ctx, "FindOne", func(tx *gorm.DB) *gorm.DB {
		return g.db.First(&groupApply, "id=?", id)
	})
	if err != nil {
		return nil, errors.Wrap(err, "FindOne")
	}
	return groupApply, nil
}

func (g *GroupApplyRepository) Insert(ctx context.Context, data *model.GroupApply) (int64, error) {
	err := g.db.Wrap(ctx, "Insert", func(tx *gorm.DB) *gorm.DB {
		return g.db.Create(&data)
	})
	if err != nil {
		return 0, errors.Wrap(err, "Insert")
	}
	return data.ID, err
}

func (g *GroupApplyRepository) ListApplyByGroupId(ctx context.Context, groupId []int64) ([]*model.GroupApply, error) {
	result := make([]*model.GroupApply, 0)
	err := g.db.Wrap(ctx, "ListApplyByGroupId", func(tx *gorm.DB) *gorm.DB {
		return g.db.Find(&result, "group_id IN ? AND status=?", groupId, GroupApplyWaitStatus)
	})
	if err != nil {
		return nil, errors.Wrap(err, "ListApplyByGroupId")
	}
	return result, nil
}

func (g *GroupApplyRepository) HandleApply(ctx context.Context, applyId int64, status string) error {
	if status == GroupApplyRejectStatus {
		err := g.db.Wrap(ctx, "HandleApply", func(tx *gorm.DB) *gorm.DB {
			return g.db.Model(&model.GroupApply{}).Where("id=?", applyId).Update("status=?", status)
		})
		if err != nil {
			return errors.Wrap(err, "HandleApply")
		}
		return nil
	} else if status == GroupApplyAccpedStatus {
		_, span := mtrace.StartSpan(ctx, "HandleApply", trace.WithSpanKind(trace.SpanKindInternal))
		defer mtrace.EndSpan(span)
		sql := make([]string, 0)
		defer func() {
			span.SetAttributes(mtrace.SQLKey.String(strings.Join(sql, "; ")))
		}()

		var groupApply *model.GroupApply
		stmt := g.db.First(&groupApply, "id=?", applyId)
		sql = append(sql, g.db.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.First(&groupApply, "id=?", applyId)
		}))
		if stmt.Error != nil {
			return stmt.Error
		}
		err := g.db.Transaction(func(tx *gorm.DB) error {
			stmt := tx.Model(&model.GroupApply{}).Where("id=?", applyId).Update("status=?", status)
			sql = append(sql, tx.ToSQL(func(tx *gorm.DB) *gorm.DB {
				return tx.Model(&model.GroupApply{}).Where("id=?", applyId).Update("status=?", status)
			}))
			if stmt.Error != nil {
				return stmt.Error
			}
			session := &model.UserSession{
				UserId: groupApply.UserId,
				ToId:   groupApply.GroupId,
				Kind:   "group",
			}
			stmt = tx.Create(session)
			sql = append(sql, tx.ToSQL(func(tx *gorm.DB) *gorm.DB {
				return tx.Create(session)
			}))
			if stmt.Error != nil {
				return stmt.Error
			}
			member := &model.GroupMember{
				GroupId:   groupApply.GroupId,
				UserId:    groupApply.UserId,
				SessionId: session.ID,
			}
			stmt = tx.Create(member)
			sql = append(sql, tx.ToSQL(func(tx *gorm.DB) *gorm.DB {
				return tx.Create(member)
			}))
			if stmt.Error != nil {
				return stmt.Error
			}
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "HandleApply")
		}
		return nil
	} else {
		return errors.New("not supported status")
	}
}

func (g *GroupApplyRepository) ApplyExists(ctx context.Context, groupId int64, userId int64) (bool, error) {
	var resp *model.GroupApply
	err := g.db.Wrap(ctx, "ApplyExists", func(tx *gorm.DB) *gorm.DB {
		return g.db.First(&resp, "group_id=? AND user_id=?", groupId, userId)
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, errors.Wrap(err, "ApplyExists")
	}
	return true, nil
}
