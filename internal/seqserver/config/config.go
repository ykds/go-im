package config

import (
	"go-im/internal/common/jwt"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/mprometheus"
	"go-im/internal/pkg/mtrace"
	"go-im/internal/pkg/redis"
	"os"

	"gopkg.in/yaml.v3"
)

type HttpServer struct {
	Addr string `yaml:"addr"`
}

type Config struct {
	Debug      bool               `yaml:"debug"`
	Pprof      bool               `yaml:"pprof"`
	Server     HttpServer         `yaml:"http"`
	JWT        jwt.Config         `yaml:"jwt"`
	Log        log.Config         `yaml:"log"`
	Trace      mtrace.Config      `yaml:"trace"`
	Redis      redis.Config       `yaml:"redis"`
	Prometheus mprometheus.Config `yaml:"prometheus"`
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
