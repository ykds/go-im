package main

import (
	"flag"
	"go-im/api/user"
	"go-im/internal/common/jwt"
	"go-im/internal/common/middleware/mgrpc"
	"go-im/internal/pkg/db"
	"go-im/internal/pkg/etcd"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/mkafka"
	"go-im/internal/pkg/mprometheus"
	"go-im/internal/pkg/mtrace"
	"go-im/internal/pkg/redis"
	"go-im/internal/user/config"
	"go-im/internal/user/server"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var cfg = flag.String("c", "./config.yaml", "")

func main() {
	flag.Parse()

	c := config.ParseConfig(*cfg)

	log.InitLogger(c.Log)
	defer log.Close()
	jwt.Init(c.JWT)
	mtrace.InitTelemetry(c.Trace)
	cli := etcd.NewClient(c.Server.Etcd)
	if cli != nil {
		err := cli.Register(c.Server.Etcd.Key, c.Server.Addr)
		if err != nil {
			panic(err)
		}
		defer cli.Close()
		defer cli.UnRegister(c.Server.Etcd.Key)
	}

	rdb := redis.NewRedis(c.Redis)
	defer rdb.Close()
	db := db.NewDB(c.Mysql)
	defer db.Close()
	kafkaWriter := mkafka.NewProducer(c.Kafka)
	defer kafkaWriter.Close()

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

	svc := server.NewServer(rdb, db, kafkaWriter)

	grpcSvc := grpc.NewServer(
		grpc.KeepaliveParams(
			keepalive.ServerParameters{
				MaxConnectionIdle: 5 * time.Minute,
				Time:              10 * time.Second,
				Timeout:           2 * time.Second,
			}),
		grpc.ChainUnaryInterceptor(mgrpc.UnaryServerRecovery(), mgrpc.UnaryServerTrace()),
		grpc.ChainStreamInterceptor(mgrpc.StreamServerRecovery(), mgrpc.StreamServerTrace()),
	)
	user.RegisterUserServer(grpcSvc, svc)

	if c.Server.Addr == "" {
		c.Server.Addr = "0.0.0.0:8000"
	}
	listen, err := net.Listen("tcp", c.Server.Addr)
	if err != nil {
		panic(err)
	}

	done := make(chan struct{})
	signals := make(chan os.Signal, 1)

	go func() {
		grpcSvc.Serve(listen)
		done <- struct{}{}
	}()

	log.Infof("user rpc server listening on %s", c.Server.Addr)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-signals:
	case <-done:
	}

	log.Infof("user rpc server shutdown.")

	grpcSvc.Stop()
}
