package config

import (
	"go-im/internal/common/jwt"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/mprometheus"
	"go-im/internal/pkg/mtrace"
	"go-im/internal/pkg/server/http"
	"go-im/internal/pkg/server/rpc"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Debug         bool               `json:"debug" yaml:"debug"`
	Pprof         bool               `json:"pprof" yaml:"pprof"`
	Server        http.Config        `yaml:"http"`
	JWT           jwt.Config         `yaml:"jwt"`
	Log           log.Config         `yaml:"log"`
	Trace         mtrace.Config      `yaml:"trace"`
	UserClient    rpc.ClientConfig   `yaml:"user_client"`
	MessageClient rpc.ClientConfig   `yaml:"message_client"`
	Prometheus    mprometheus.Config `jyaml:"prometheus"`
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
