package main

import (
	"flag"
	"go-im/api/message"
	"go-im/internal/common/middleware/mgrpc"
	"go-im/internal/message/config"
	"go-im/internal/message/server"
	"go-im/internal/pkg/db"
	"go-im/internal/pkg/etcd"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/mkafka"
	"go-im/internal/pkg/mtrace"
	"go-im/internal/pkg/redis"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var cfg = flag.String("c", "./config.yaml", "")

func main() {
	flag.Parse()

	c := config.ParseConfig(*cfg)

	log.InitLogger(c.Log)
	mtrace.InitTelemetry(c.Trace)
	cli := etcd.NewClient(c.Etcd)
	if cli != nil {
		err := cli.Register(c.Etcd.Key, c.Etcd.Addr)
		if err != nil {
			panic(err)
		}
		defer cli.Close()
		defer cli.UnRegister(c.Etcd.Key)
	}

	rdb := redis.NewRedis(c.Redis)
	defer rdb.Close()
	db := db.NewDB(c.Mysql)
	defer db.Close()
	kafkaWriter := mkafka.NewProducer(c.Kafka)
	defer kafkaWriter.Close()

	userAddr := c.UserClient.ParseAddr()
	svc := server.NewServer(c, rdb, db, kafkaWriter, userAddr)

	grpcSvc := grpc.NewServer(
		grpc.KeepaliveParams(
			keepalive.ServerParameters{
				MaxConnectionIdle: 5 * time.Minute,
				Time:              10 * time.Second,
				Timeout:           2 * time.Second,
			}),
		grpc.ChainUnaryInterceptor(mgrpc.UnaryServerTrace()),
		grpc.ChainStreamInterceptor(mgrpc.StreamServerTrace()),
	)
	message.RegisterMessageServer(grpcSvc, svc)

	if c.Addr == "" {
		c.Addr = "0.0.0.0:8001"
	}
	listen, err := net.Listen("tcp", c.Addr)
	if err != nil {
		panic(err)
	}

	done := make(chan struct{})
	signals := make(chan os.Signal, 1)

	go func() {
		grpcSvc.Serve(listen)
		done <- struct{}{}
	}()

	log.Infof("message rpc server listening on %s", c.Addr)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-signals:
	case <-done:
	}

	log.Infof("message rpc server shutdown.")

	grpcSvc.Stop()
}
