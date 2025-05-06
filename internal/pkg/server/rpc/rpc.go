package rpc

import (
	"context"
	"go-im/internal/common/middleware/mgrpc"
	"go-im/internal/pkg/etcd"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type ServerConfig struct {
	Addr              string      `yaml:"addr"`
	Etcd              etcd.Config `yaml:"etcd"`
	HeartBeatInterval int         `yaml:"heart_beat_interval"`
	HeartBeatTimeout  int         `yaml:"heart_beat_timeout"`
	MaxConnectionIdle int         `yaml:"max_connection_idle"`
	Trace             bool
}

func NewGrpcServer(c *ServerConfig) *grpc.Server {
	var (
		hbInterval       int = 10
		hbTimeout        int = 3
		maxConnectIdle   int = 10
		unaryInterceptor     = []grpc.UnaryServerInterceptor{
			mgrpc.UnaryServerRecovery(),
		}
		streamInterceptor = []grpc.StreamServerInterceptor{
			mgrpc.StreamServerRecovery(),
		}
	)
	if c.HeartBeatInterval != 0 {
		hbInterval = c.HeartBeatInterval
	}
	if c.HeartBeatTimeout != 0 {
		hbTimeout = c.HeartBeatTimeout
	}
	if c.MaxConnectionIdle != 0 {
		maxConnectIdle = c.MaxConnectionIdle
	}
	if c.Trace {
		unaryInterceptor = append(unaryInterceptor, mgrpc.UnaryServerTrace())
		streamInterceptor = append(streamInterceptor, mgrpc.StreamServerTrace())
	}
	return grpc.NewServer(
		grpc.KeepaliveParams(
			keepalive.ServerParameters{
				MaxConnectionIdle: time.Duration(maxConnectIdle) * time.Minute,
				Time:              time.Duration(hbInterval) * time.Second,
				Timeout:           time.Duration(hbTimeout) * time.Second,
			}),
		grpc.ChainUnaryInterceptor(unaryInterceptor...),
		grpc.ChainStreamInterceptor(streamInterceptor...),
	)
}

type ClientConfig struct {
	Type string      `yaml:"type"`
	Addr string      `yaml:"addr"`
	Etcd etcd.Config `yaml:"etcd"`
}

func (g ClientConfig) ParseAddr() string {
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
