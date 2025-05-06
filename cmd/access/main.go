package main

import (
	"context"
	"flag"
	"go-im/api/access"
	"go-im/internal/access/config"
	"go-im/internal/access/server"
	"go-im/internal/common/jwt"
	"go-im/internal/common/middleware/mhttp"
	"go-im/internal/pkg/etcd"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/mpprof"
	"go-im/internal/pkg/mtrace"
	"go-im/internal/pkg/server/rpc"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var cfg = flag.String("c", "./config.yaml", "")

func main() {
	flag.Parse()

	c := config.ParseConfig(*cfg)

	log.InitLogger(c.Log)
	defer log.Close()
	jwt.Init(c.JWT)
	mtrace.InitTelemetry(c.Trace)

	if c.Pprof {
		mpprof.RegisterPprof()
	}

	wsServer := server.NewServer(c)

	// rpc server
	c.RPC.Trace = c.Trace.Enable
	grpcSvc := rpc.NewGrpcServer(&c.RPC)
	access.RegisterAccessServer(grpcSvc, wsServer)
	if c.RPC.Addr == "" {
		c.RPC.Addr = "0.0.0.0:8012"
	}
	listen, err := net.Listen("tcp", c.RPC.Addr)
	if err != nil {
		panic(err)
	}
	if c.RPC.Etcd.Key != "" && c.RPC.Etcd.Addr != "" {
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

	// http server
	if c.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	gin.DefaultWriter = io.Discard
	engine := gin.New()
	engine.Use(gin.RecoveryWithWriter(log.Output()), mhttp.Cors())
	if c.Trace.Enable {
		engine.Use(mhttp.Trace())
	}
	if c.Prometheus.Enable {
		engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}
	engine.GET("/ws", mhttp.AuthMiddleware(), wsServer.Handler)
	if c.Http.Addr == "" {
		c.Http.Addr = "0.0.0.0:8002"
	}
	svc := http.Server{
		Addr:    c.Http.Addr,
		Handler: engine,
	}

	done := make(chan struct{}, 2)
	signals := make(chan os.Signal, 1)

	go func() {
		svc.ListenAndServe()
		done <- struct{}{}
	}()
	go func() {
		grpcSvc.Serve(listen)
		done <- struct{}{}
	}()

	log.Infof("access http server listening on %s", c.Http.Addr)
	log.Infof("access rpc server listening on %s", c.RPC.Addr)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-signals:
	case <-done:
	}

	log.Infof("access server shutdown.")

	svc.Shutdown(context.TODO())
	wsServer.Stop()
}
