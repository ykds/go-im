package main

import (
	"context"
	"flag"
	"go-im/internal/access/config"
	"go-im/internal/access/server"
	"go-im/internal/common/jwt"
	"go-im/internal/common/middleware/mhttp"
	"go-im/internal/pkg/log"
	"go-im/internal/pkg/mpprof"
	"go-im/internal/pkg/mtrace"
	"io"
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
	defer log.Close()
	jwt.Init(c.JWT)
	mtrace.InitTelemetry(c.Trace)

	wsServer := server.NewServer(c)

	if c.Server.Addr == "" {
		c.Server.Addr = "0.0.0.0:8002"
	}
	if c.Server.Debug {
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
	if c.Server.Debug {
		pprof.Register(engine)
	}
	if c.Prometheus.Enable {
		engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}
	if c.Server.Pprof {
		mpprof.RegisterPprof()
	}
	engine.GET("/ws", mhttp.AuthMiddleware(), wsServer.Handler)
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

	log.Infof("access server listening on %s", c.Server.Addr)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-signals:
	case <-done:
	}

	log.Infof("access server shutdown.")

	svc.Shutdown(context.TODO())
	wsServer.Stop()
}
