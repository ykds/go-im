package mprometheus

import (
	"go-im/internal/pkg/redis"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/extra/redisprometheus/v9"
	"gorm.io/gorm"
	gormp "gorm.io/plugin/prometheus"
)

type Config struct {
	Listen   string `json:"listen" yaml:"listen"`
	Addr     string `json:"addr" yaml:"addr"`
	User     string `json:"user" yaml:"user"`
	Password string `json:"password" yaml:"password"`
	Enable   bool   `json:"enable" yaml:"enable"`
}

func GormPrometheus(c *Config, db *gorm.DB, dbName string) {
	db.Use(gormp.New(gormp.Config{
		DBName:          dbName,
		RefreshInterval: 15,
		PushAddr:        c.Addr,
		PushUser:        c.User,
		PushPassword:    c.Password,
	}))
}

func RedisPrometheus(c *Config, rdb *redis.Redis, namespace, subsystem string) prometheus.Collector {
	return redisprometheus.NewCollector(namespace, subsystem, rdb)
}
