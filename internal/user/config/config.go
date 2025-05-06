package config

import (
	"go-im/internal/common/jwt"
	"go-im/internal/pkg/db"
	"go-im/internal/pkg/kafka"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/mprometheus"
	"go-im/internal/pkg/mtrace"
	"go-im/internal/pkg/redis"
	"go-im/internal/pkg/server/rpc"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Pprof        bool               `yaml:"pprof"`
	RPC          rpc.ServerConfig   `yaml:"rpc"`
	Mysql        db.Config          `yaml:"mysql"`
	Redis        redis.Config       `yaml:"redis"`
	Kafka        kafka.Config       `yaml:"kafka"`
	JWT          jwt.Config         `yaml:"jwt"`
	Log          log.Config         `yaml:"log"`
	Trace        mtrace.Config      `yaml:"trace"`
	Prometheus   mprometheus.Config `yaml:"prometheus"`
	AccessClient rpc.ClientConfig   `yaml:"access_client"`
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
