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

type GroupMemberRepository struct {
	db *db.DB
}

func NewGroupMemberRepository(db *db.DB) *GroupMemberRepository {
	return &GroupMemberRepository{db}
}

func (g *GroupMemberRepository) MemberCount(ctx context.Context, groupId int64) (int64, error) {
	var count int64
	err := g.db.Wrap(ctx, "MemberCount", func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&model.GroupMember{}).Where("group_id=?", groupId).Count(&count)
	})
	if err != nil {
		return 0, errors.Wrap(err, "MemberCount")
	}
	return count, nil
}

func (g *GroupMemberRepository) InviteMember(ctx context.Context, groupId int64, userId []int64) error {
	_, span := mtrace.StartSpan(ctx, "InviteMember", trace.WithSpanKind(trace.SpanKindInternal))
	defer mtrace.EndSpan(span)
	sql := make([]string, 0)
	defer func() {
		span.SetAttributes(mtrace.SQLKey.String(strings.Join(sql, "; ")))
	}()
	err := g.db.Transaction(func(tx *gorm.DB) error {
		sessions := make([]*model.UserSession, 0, len(userId))
		for _, id := range userId {
			sessions = append(sessions, &model.UserSession{
				UserId: id,
				ToId:   groupId,
				Kind:   "group",
			})
		}
		stmt := tx.Create(sessions)
		sql = append(sql, tx.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Create(sessions)
		}))
		if stmt.Error != nil {
			return stmt.Error
		}
		userSessionMap := make(map[int64]int64, len(userId))
		for _, session := range sessions {
			userSessionMap[session.UserId] = session.ID
		}
		members := make([]*model.GroupMember, 0, len(userId))
		for _, id := range userId {
			members = append(members, &model.GroupMember{
				GroupId:   groupId,
				UserId:    id,
				SessionId: userSessionMap[id],
			})
		}
		stmt = tx.Create(members)
		sql = append(sql, tx.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Create(members)
		}))
		if stmt.Error != nil {
			return stmt.Error
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "InviteMember")
	}
	return nil
}

func (g *GroupMemberRepository) IsMember(ctx context.Context, groupId int64, userId int64) (bool, error) {
	var resp *model.GroupMember
	err := g.db.Wrap(ctx, "IsMember", func(tx *gorm.DB) *gorm.DB {
		return tx.First(&resp, "group_id=? AND user_id=?", groupId, userId)
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, errors.Wrap(err, "IsMember")
	}
	return true, nil
}

func (g *GroupMemberRepository) RemvoeMember(ctx context.Context, groupId int64, userId int64) error {
	_, span := mtrace.StartSpan(ctx, "RemvoeMember", trace.WithSpanKind(trace.SpanKindInternal))
	defer mtrace.EndSpan(span)
	sql := make([]string, 0)
	defer func() {
		span.SetAttributes(mtrace.SQLKey.String(strings.Join(sql, "; ")))
	}()
	err := g.db.Transaction(func(tx *gorm.DB) error {
		stmt := tx.Delete(&model.GroupMember{}, "group_id=? AND user_id=?", groupId, userId)
		sql = append(sql, tx.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Delete(&model.GroupMember{}, "group_id=? AND user_id=?", groupId, userId)
		}))
		if stmt.Error != nil {
			return stmt.Error
		}
		stmt = tx.Delete(&model.UserSession{}, "group_id=? AND user_id=?", groupId, userId)
		sql = append(sql, tx.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Delete(&model.UserSession{}, "group_id=? AND user_id=?", groupId, userId)
		}))
		if stmt.Error != nil {
			return stmt.Error
		}
		return nil
	})
	if err != nil {
		errors.Wrap(err, "RemvoeMember")
	}
	return nil
}

func (g *GroupMemberRepository) ListGroupByUserId(ctx context.Context, userId int64) ([]int64, error) {
	resp := make([]*model.GroupMember, 0)
	err := g.db.Wrap(ctx, "ListGroupByUserId", func(tx *gorm.DB) *gorm.DB {
		return tx.Find(&resp, "user_id=?", userId)
	})
	if err != nil {
		return nil, errors.Wrap(err, "ListGroupByUserId")
	}
	ids := make([]int64, 0, len(resp))
	for _, g := range resp {
		ids = append(ids, g.GroupId)
	}
	return ids, nil
}

func (g *GroupMemberRepository) ListMember(ctx context.Context, groupId int64) ([]*model.GroupMember, error) {
	var resp []*model.GroupMember
	err := g.db.Wrap(ctx, "ListMember", func(tx *gorm.DB) *gorm.DB {
		return tx.Find(&resp, "group_id=?", groupId)
	})
	if err != nil {
		return nil, errors.Wrap(err, "ListMember")
	}
	return resp, nil
}
