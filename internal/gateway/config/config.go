package config

import (
	"context"
	"go-im/internal/common/jwt"
	"go-im/internal/pkg/etcd"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/mtrace"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Addr  string `json:"addr" yaml:"addr"`
	Debug bool   `json:"debug" yaml:"debug"`
}

type GrpcClient struct {
	Type string      `json:"type" yaml:"type"`
	Addr string      `json:"addr" yaml:"addr"`
	Etcd etcd.Config `json:"etcd" yaml:"etcd"`
}

func (g GrpcClient) ParseAddr() string {
	switch g.Type {
	case "etcd":
		cli2 := etcd.NewClient(g.Etcd)
		if cli2 == nil {
			panic("etcd addr empty")
		}
		ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
		resp, err := cli2.Get(ctx, g.Etcd.Key)
		cancel()
		if err != nil {
			cli2.Close()
			panic(err)
		}
		if len(resp.Kvs) > 0 {
			cli2.Close()
			return string(resp.Kvs[0].Value)
		}
		cli2.Close()
	case "direct":
		return g.Addr
	default:
		panic("empty rpc config")
	}
	return ""
}

type Config struct {
	Server        ServerConfig  `json:"server" yaml:"server"`
	JWT           jwt.Config    `json:"jwt" yaml:"jwt"`
	Log           log.Config    `json:"log" yaml:"log"`
	Trace         mtrace.Config `json:"trace" yaml:"trace"`
	UserClient    GrpcClient    `json:"user_client" yaml:"user_client"`
	MessageClient GrpcClient    `json:"message_client" yaml:"message_client"`
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
