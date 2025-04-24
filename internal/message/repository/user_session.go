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
	"gorm.io/gorm/clause"
)

type UserSessionRepository struct {
	db *db.DB
}

func NewUserSessionRepository(db *db.DB) *UserSessionRepository {
	return &UserSessionRepository{db}
}

func (u *UserSessionRepository) Delete(ctx context.Context, id int64) error {
	err := u.db.Wrap(ctx, "Delete", func(tx *gorm.DB) *gorm.DB {
		return tx.Delete(&model.UserSession{}, "id=?", id)
	})
	if err != nil {
		return errors.Wrap(err, "Delete")
	}
	return nil
}

func (u *UserSessionRepository) GetUserSession(ctx context.Context, userId int64, to int64) (*model.UserSession, error) {
	var resp *model.UserSession
	err := u.db.Wrap(ctx, "GetUserSession", func(tx *gorm.DB) *gorm.DB {
		return tx.First(&resp, "user_id=? AND to_id=?", userId, to)
	})
	if err != nil {
		return nil, errors.Wrap(err, "GetUserSession")
	}
	return resp, nil
}

func (u *UserSessionRepository) ListUserSession(ctx context.Context, userId int64) ([]*model.UserSession, error) {
	var resp []*model.UserSession
	err := u.db.Wrap(ctx, "ListUserSession", func(tx *gorm.DB) *gorm.DB {
		return tx.Find(&resp, "user_id=?", userId)
	})
	if err != nil {
		return nil, errors.Wrap(err, "GetUserSession")
	}
	return resp, nil
}

func (u *UserSessionRepository) UpdateSessionSeq(ctx context.Context, id int64, seq int64) error {
	err := u.db.Wrap(ctx, "UpdateSessionSeq", func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&model.UserSession{}).Where("id=? AND seq <?", id, seq).Update("seq=?", seq)
	})
	if err != nil {
		return errors.Wrap(err, "UpdateSessionSeq")
	}
	return nil
}

func (u *UserSessionRepository) Upsert(ctx context.Context, s *model.UserSession) error {
	_, span := mtrace.StartSpan(ctx, "Upsert", trace.WithSpanKind(trace.SpanKindInternal))
	defer mtrace.EndSpan(span)
	sql := make([]string, 0)
	defer func() {
		span.SetAttributes(mtrace.SQLKey.String(strings.Join(sql, "; ")))
	}()
	err := u.db.Transaction(func(tx *gorm.DB) error {
		stmt := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoNothing: true,
		}).Create(&s)
		sql = append(sql, tx.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}},
				DoNothing: true,
			}).Create(&s)
		}))
		if stmt.Error != nil {
			return stmt.Error
		}
		s2 := &model.UserSession{
			Kind:   s.Kind,
			UserId: s.ToId,
			ToId:   s.UserId,
		}
		stmt = tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoNothing: true,
		}).Create(&s2)
		sql = append(sql, tx.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}},
				DoNothing: true,
			}).Create(&s2)
		}))
		if stmt.Error != nil {
			return stmt.Error
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "Upsert")
	}
	return nil
}

func (u *UserSessionRepository) Create(ctx context.Context, s *model.UserSession) (int64, error) {
	_, span := mtrace.StartSpan(ctx, "Create", trace.WithSpanKind(trace.SpanKindInternal))
	defer mtrace.EndSpan(span)
	sql := make([]string, 0)
	defer func() {
		span.SetAttributes(mtrace.SQLKey.String(strings.Join(sql, "; ")))
	}()
	stmt := u.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoNothing: true,
	}).Create(&s)
	sql = append(sql, u.db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoNothing: true,
		}).Create(&s)
	}))
	if stmt.Error != nil {
		return 0, errors.Wrap(stmt.Error, "Create")
	}
	var resp *model.UserSession
	stmt = u.db.First(&resp, "user_id=? AND to_id=? AND kind=?", s.UserId, s.ToId, s.Kind)
	sql = append(sql, u.db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.First(&resp, "user_id=? AND to_id=? AND kind=?", s.UserId, s.ToId, s.Kind)
	}))
	if stmt.Error != nil {
		return 0, errors.Wrap(stmt.Error, "Create")
	}
	return resp.ID, nil
}
