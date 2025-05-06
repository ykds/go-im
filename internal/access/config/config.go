package config

import (
	"go-im/internal/common/jwt"
	"go-im/internal/pkg/kafka"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/mprometheus"
	"go-im/internal/pkg/mtrace"
	"go-im/internal/pkg/redis"
	"go-im/internal/pkg/server/http"
	"go-im/internal/pkg/server/rpc"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Debug         bool               `yaml:"debug"`
	Pprof         bool               `yaml:"pprof"`
	Http          http.Config        `yaml:"http"`
	RPC           rpc.ServerConfig   `yaml:"rpc"`
	Redis         redis.Config       `yaml:"redis"`
	Kafka         kafka.Config       `yaml:"kafka"`
	JWT           jwt.Config         `yaml:"jwt"`
	Log           log.Config         `yaml:"log"`
	Trace         mtrace.Config      `yaml:"trace"`
	UserClient    rpc.ClientConfig   `yaml:"user_client"`
	MessageClient rpc.ClientConfig   `yaml:"message_client"`
	Prometheus    mprometheus.Config `yaml:"prometheus"`
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
