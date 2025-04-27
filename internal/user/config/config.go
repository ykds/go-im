package config

import (
	"go-im/internal/common/jwt"
	"go-im/internal/pkg/db"
	"go-im/internal/pkg/etcd"
	"go-im/internal/pkg/kafka"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/mprometheus"
	"go-im/internal/pkg/mtrace"
	"go-im/internal/pkg/redis"
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Addr string      `json:"addr" yaml:"addr"`
	Etcd etcd.Config `json:"etcd" yaml:"etcd"`
}

type Config struct {
	Server     ServerConfig       `json:"server" yaml:"server"`
	Mysql      db.Config          `json:"mysql" yaml:"mysql"`
	Redis      redis.Config       `json:"redis" yaml:"redis"`
	Kafka      kafka.Config       `json:"kafka" yaml:"kafka"`
	JWT        jwt.Config         `json:"jwt" yaml:"jwt"`
	Log        log.Config         `json:"log" yaml:"log"`
	Trace      mtrace.Config      `json:"trace" yaml:"trace"`
	Prometheus mprometheus.Config `json:"prometheus" yaml:"prometheus"`
}

func ParseConfig(file string) *Config {
	content, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}
	cfg := &Config{}
	err = yaml.Unmarshal(content, cfg)
	if err != nil {
		panic(err)
	}
	return cfg
}
