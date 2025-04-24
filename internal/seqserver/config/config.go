package config

import (
	"go-im/internal/common/jwt"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/mtrace"
	"go-im/internal/pkg/redis"
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Addr  string `json:"addr" yaml:"addr"`
	Debug bool   `json:"debug" yaml:"debug"`
}

type Config struct {
	Server ServerConfig  `json:"server" yaml:"server"`
	JWT    jwt.Config    `json:"jwt" yaml:"jwt"`
	Log    log.Config    `json:"log" yaml:"log"`
	Trace  mtrace.Config `json:"trace" yaml:"trace"`
	Redis  redis.Config  `json:"redis" yaml:"redis"`
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
