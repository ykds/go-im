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

type GroupRepository struct {
	db *db.DB
}

func NewGroupRepository(db *db.DB) *GroupRepository {
	return &GroupRepository{db}
}

func (g *GroupRepository) FindOne(ctx context.Context, id int64) (*model.Group, error) {
	var resp *model.Group
	err := g.db.Wrap(ctx, "FindOne", func(tx *gorm.DB) *gorm.DB {
		return tx.First(&resp, "id=?", id)
	})
	if err != nil {
		return nil, errors.Wrap(err, "FindOne")
	}
	return resp, nil
}

func (g *GroupRepository) FindOneByGroupNo(ctx context.Context, groupNo int64) (*model.Group, error) {
	var resp *model.Group
	err := g.db.Wrap(ctx, "FindOneByGroupNo", func(tx *gorm.DB) *gorm.DB {
		return g.db.First(&resp, "group_no=?", groupNo)
	})
	if err != nil {
		return nil, errors.Wrap(err, "FindOne")
	}
	return resp, nil
}

func (g *GroupRepository) Update(ctx context.Context, newData *model.Group) error {
	err := g.db.Wrap(ctx, "Update", func(tx *gorm.DB) *gorm.DB {
		return g.db.Updates(newData)
	})
	if err != nil {
		return errors.Wrap(err, "Update")
	}
	return nil
}

func (g *GroupRepository) ListGroupByOwnerId(ctx context.Context, ownerId int64) ([]*model.Group, error) {
	var resp []*model.Group
	err := g.db.Wrap(ctx, "ListGroupByOwnerId", func(tx *gorm.DB) *gorm.DB {
		return g.db.Find(&resp, "owner_id=?", ownerId)
	})
	if err != nil {
		return nil, errors.Wrap(err, "ListGroupByOwnerId")
	}
	return resp, nil
}

func (g *GroupRepository) CreateGroup(ctx context.Context, group *model.Group) (int64, error) {
	_, span := mtrace.StartSpan(ctx, "CreateGroup", trace.WithSpanKind(trace.SpanKindInternal))
	defer mtrace.EndSpan(span)
	sql := make([]string, 0)
	defer func() {
		span.SetAttributes(mtrace.SQLKey.String(strings.Join(sql, "; ")))
	}()
	err := g.db.Transaction(func(tx *gorm.DB) error {
		stmt := tx.Create(&group)
		sql = append(sql, tx.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Create(&group)
		}))
		if stmt.Error != nil {
			return stmt.Error
		}
		session := &model.UserSession{
			UserId: group.OwnerId,
			ToId:   group.ID,
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
			GroupId:   group.ID,
			UserId:    group.OwnerId,
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
		return 0, errors.Wrap(err, "CreateGroup")
	}
	return group.ID, nil
}

func (g *GroupRepository) ListGroupById(ctx context.Context, ids []int64) ([]*model.Group, error) {
	var resp []*model.Group
	err := g.db.Wrap(ctx, "ListGroupById", func(tx *gorm.DB) *gorm.DB {
		return g.db.Find(&resp, "id IN ?", ids)
	})
	if err != nil {
		return nil, errors.Wrap(err, "ListGroupById")
	}
	return resp, nil
}

func (g *GroupRepository) UpdateGroup(ctx context.Context, id int64, group *model.Group) error {
	err := g.db.Wrap(ctx, "UpdateGroup", func(tx *gorm.DB) *gorm.DB {
		return g.db.Where("id=?", id).Updates(group)
	})
	if err != nil {
		return errors.Wrap(err, "UpdateGroup")
	}
	return nil
}

func (g *GroupRepository) DismissGroup(ctx context.Context, id int64) error {
	_, span := mtrace.StartSpan(ctx, "DismissGroup", trace.WithSpanKind(trace.SpanKindInternal))
	defer mtrace.EndSpan(span)
	sql := make([]string, 0)
	defer func() {
		span.SetAttributes(mtrace.SQLKey.String(strings.Join(sql, "; ")))
	}()
	err := g.db.Transaction(func(tx *gorm.DB) error {
		stmt := tx.Delete(&model.GroupMember{}, "group_id=?", id)
		sql = append(sql, tx.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Delete(&model.GroupMember{}, "group_id=?", id)
		}))
		if stmt.Error != nil {
			return stmt.Error
		}
		stmt = tx.Delete(&model.Group{}, "id=?", id)
		sql = append(sql, tx.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Delete(&model.Group{}, "id=?", id)
		}))
		if stmt.Error != nil {
			return stmt.Error
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "DismissGroup")
	}
	return nil
}
