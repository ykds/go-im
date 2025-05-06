package main

import (
	"context"
	"flag"
	"go-im/internal/common/jwt"
	"go-im/internal/common/middleware/mhttp"
	"go-im/internal/gateway/config"
	"go-im/internal/gateway/logic"
	"go-im/internal/gateway/server"
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

	if c.Pprof {
		mpprof.RegisterPprof()
	}

	gin.DefaultWriter = io.Discard
	engine := gin.New()
	engine.Use(gin.Recovery(), gin.RecoveryWithWriter(log.Output()), mhttp.Cors())
	if c.Trace.Enable {
		engine.Use(mhttp.Trace())
	}
	if c.Debug {
		pprof.Register(engine)
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	if c.Prometheus.Enable {
		engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	engine.Static("/static", c.Server.Static)

	api := engine.Group("/api")

	s := server.NewServer(c)
	friedApi := logic.NewFriendApi(s)
	friedApi.RegisterRouter(api)

	groupApi := logic.NewGroupApi(s)
	groupApi.RegisterRouter(api)

	messageApi := logic.NewMessageApi(s)
	messageApi.RegisterRouter(api)

	userApi := logic.NewUserApi(s)
	userApi.RegisterRouter(api)

	uploadApi := logic.NewUploadApi(s)
	uploadApi.RegisterRouter(api)

	if c.Server.Addr == "" {
		c.Server.Addr = "0.0.0.0:9000"
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

	log.Infof("gateway server listening on %s", c.Server.Addr)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-signals:
	case <-done:
	}

	log.Infof("gateway server shutdown.")

	svc.Shutdown(context.TODO())
}
