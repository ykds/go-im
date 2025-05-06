package main

import (
	"flag"
	"go-im/api/access"
	"go-im/api/user"
	"go-im/internal/common/jwt"
	"go-im/internal/common/middleware/mgrpc"
	"go-im/internal/pkg/db"
	"go-im/internal/pkg/etcd"
	"go-im/internal/pkg/kafka"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/mpprof"
	"go-im/internal/pkg/mprometheus"
	"go-im/internal/pkg/mtrace"
	"go-im/internal/pkg/redis"
	"go-im/internal/pkg/server/rpc"
	"go-im/internal/user/config"
	"go-im/internal/user/server"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var cfg = flag.String("c", "./config.yaml", "")

func main() {
	flag.Parse()

	c := config.ParseConfig(*cfg)

	log.InitLogger(c.Log)
	defer log.Close()
	jwt.Init(c.JWT)
	mtrace.InitTelemetry(c.Trace)

	rdb := redis.NewRedis(c.Redis)
	defer rdb.Close()
	db := db.NewDB(c.Mysql)
	defer db.Close()

	var kafkaWriter *kafka.Writer
	if c.Kafka.Enable {
		kafkaWriter := kafka.NewProducer(c.Kafka)
		defer kafkaWriter.Close()
	}

	if c.Prometheus.Enable {
		mprometheus.GormPrometheus(&c.Prometheus, db.DB, "im")
		prometheus.MustRegister(mprometheus.RedisPrometheus(&c.Prometheus, rdb, "go-im", "user"))
		http.Handle("/metrics", promhttp.Handler())
		go func() {
			err := http.ListenAndServe(c.Prometheus.Listen, nil)
			if err != nil {
				panic(err)
			}
		}()
	}

	if c.Pprof {
		mpprof.RegisterPprof()
	}

	// access rpc client init
	var accessClient access.AccessClient
	accessAddr := c.AccessClient.ParseAddr()
	if accessAddr == "" && !c.Kafka.Enable {
		panic("kafka or access rpc must set")
	}
	if accessAddr != "" {
		opts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
		if c.Trace.Enable {
			opts = append(opts, grpc.WithChainUnaryInterceptor(mgrpc.UnaryClientTrace()))
			opts = append(opts, grpc.WithChainStreamInterceptor(mgrpc.StreamClientTrace()))
		}
		conn, err := grpc.NewClient(accessAddr, opts...)
		if err != nil {
			panic(err)
		}
		accessClient = access.NewAccessClient(conn)
	}

	svc := server.NewServer(rdb, db, kafkaWriter, accessClient)

	c.RPC.Trace = c.Trace.Enable
	grpcSvc := rpc.NewGrpcServer(&c.RPC)
	user.RegisterUserServer(grpcSvc, svc)
	if c.RPC.Addr == "" {
		c.RPC.Addr = "0.0.0.0:8000"
	}
	listen, err := net.Listen("tcp", c.RPC.Addr)
	if err != nil {
		panic(err)
	}
	if c.RPC.Etcd.Addr != "" && c.RPC.Etcd.Key != "" {
		cli := etcd.NewClient(c.RPC.Etcd)
		if cli != nil {
			err := cli.Register(c.RPC.Etcd.Key, c.RPC.Addr)
			if err != nil {
				panic(err)
			}
			defer cli.Close()
			defer cli.UnRegister(c.RPC.Etcd.Key)
		}
	}

	done := make(chan struct{})
	signals := make(chan os.Signal, 1)

	go func() {
		grpcSvc.Serve(listen)
		done <- struct{}{}
	}()

	log.Infof("user rpc server listening on %s", c.RPC.Addr)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-signals:
	case <-done:
	}

	log.Infof("user rpc server shutdown.")

	grpcSvc.Stop()
}
