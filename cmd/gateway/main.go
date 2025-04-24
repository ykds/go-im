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
	"go-im/internal/pkg/mtrace"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

var cfg = flag.String("c", "./config.yaml", "")

func main() {
	flag.Parse()

	c := config.ParseConfig(*cfg)

	log.InitLogger(c.Log)
	jwt.Init(c.JWT)
	mtrace.InitTelemetry(c.Trace)

	if c.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(gin.RecoveryWithWriter(log.Output()))
	if c.Trace.Enable {
		engine.Use(mhttp.Trace())
	}
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

	if c.Addr == "" {
		c.Addr = "0.0.0.0:8003"
	}
	svc := http.Server{
		Addr:    c.Addr,
		Handler: engine,
	}

	done := make(chan struct{})
	signals := make(chan os.Signal, 1)

	go func() {
		svc.ListenAndServe()
		done <- struct{}{}
	}()

	log.Infof("gateway server listening on %s", c.Addr)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-signals:
	case <-done:
	}

	log.Infof("gateway server shutdown.")

	svc.Shutdown(context.TODO())
}
