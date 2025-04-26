package db

import (
	"context"
	"database/sql"
	"fmt"
	"go-im/internal/pkg/mtrace"
	"net/url"
	"time"

	"go.opentelemetry.io/otel/trace"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DriverType string

const (
	Postgres DriverType = "postgres"
	MySQL    DriverType = "mysql"
)

type Config struct {
	Driver          DriverType `json:"driver" yaml:"driver"`
	Host            string     `json:"host" yaml:"host"`
	Port            string     `json:"port" yaml:"port"`
	User            string     `json:"user" yaml:"user"`
	Password        string     `json:"password" yaml:"password"`
	DbName          string     `json:"db_name" yaml:"db_name"`
	Timezone        string     `json:"timezone" yaml:"timezone"`
	MaxOpenConns    int        `json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns    int        `json:"max_idle_conns" yaml:"max_idle_conns"`
	ConnMaxLifeTime int        `json:"conn_max_life_time" yaml:"conn_max_life_time"`
}

type DB struct {
	*gorm.DB
	sqlDB *sql.DB
}

func NewDB(cfg Config) *DB {
	var (
		db  *gorm.DB
		err error
	)
	switch cfg.Driver {
	case Postgres:
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=%s",
			cfg.Host, cfg.User, cfg.Password, cfg.DbName, cfg.Port, cfg.Timezone)
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	case MySQL:
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=%s",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DbName, url.QueryEscape(cfg.Timezone))
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	default:
		panic("unsupported driver")
	}
	if err != nil {
		panic(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	maxOpenConns := cfg.MaxOpenConns
	if maxOpenConns == 0 {
		maxOpenConns = 10
	}
	maxIdleConns := cfg.MaxIdleConns
	if maxIdleConns == 0 {
		maxIdleConns = 5
	}
	connMaxLifeTime := cfg.ConnMaxLifeTime
	if cfg.ConnMaxLifeTime == 0 {
		connMaxLifeTime = 60000
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(connMaxLifeTime) * time.Millisecond)
	return &DB{DB: db, sqlDB: sqlDB}
}

func (db *DB) Debug() *DB {
	db.DB = db.DB.Debug()
	return db
}

func (db *DB) Close() error {
	return db.sqlDB.Close()
}

func (db *DB) Wrap(ctx context.Context, name string, f func(tx *gorm.DB) *gorm.DB) error {
	_, span := mtrace.StartSpan(ctx, name, trace.WithSpanKind(trace.SpanKindInternal))
	defer mtrace.EndSpan(span)
	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return f(tx)
	})
	stmt := f(db.DB)
	span.SetAttributes(mtrace.SQLKey.String(sql))
	if stmt.Error != nil {
		span.SetAttributes(mtrace.SQLError.String(stmt.Error.Error()))
	}
	return stmt.Error
}
