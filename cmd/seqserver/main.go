package main

import (
	"context"
	"flag"
	"go-im/internal/common/jwt"
	"go-im/internal/common/middleware/mhttp"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/mtrace"
	"go-im/internal/pkg/redis"
	"go-im/internal/seqserver/config"
	"go-im/internal/seqserver/logic"
	"go-im/internal/seqserver/pkg/seqserver"
	"go-im/internal/seqserver/server"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var cfg = flag.String("c", "./config.yaml", "")

func main() {
	flag.Parse()

	c := config.ParseConfig(*cfg)

	log.InitLogger(c.Log)
	jwt.Init(c.JWT)
	mtrace.InitTelemetry(c.Trace)
	rdb := redis.NewRedis(c.Redis)
	defer rdb.Close()

	if c.Server.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(gin.RecoveryWithWriter(log.Output()), mhttp.Cors())
	if c.Trace.Enable {
		engine.Use(mhttp.Trace())
	}
	if c.Server.Debug {
		pprof.Register(engine)
	}
	if c.Prometheus.Enable {
		engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}
	api := engine.Group("/api")

	seqServer := seqserver.NewRedisSeqServer(rdb)

	s := server.NewServer(seqServer)
	seqApi := logic.NewSeqApi(s)
	seqApi.RegisterRouter(api)

	if c.Server.Addr == "" {
		c.Server.Addr = "0.0.0.0:8004"
	}
	svc := http.Server{
		Addr:    c.Server.Addr,
		Handler: engine,
	}

	done := make(chan struct{})
	signals := make(chan os.Signal, 1)

	go func() {
		svc.ListenAndServe()
		done <- struct{}{}
	}()

	log.Infof("seqserver server listening on %s", c.Server.Addr)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-signals:
	case <-done:
	}

	log.Infof("seqserver server shutdown.")

	svc.Shutdown(context.TODO())
}
