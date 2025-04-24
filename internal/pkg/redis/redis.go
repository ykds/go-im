package redis

import (
	"context"
	"go-im/internal/pkg/mtrace"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	Addr string `json:"addr"`
}

type Redis struct {
	*redis.Client
}

func NewRedis(cfg Config) *Redis {
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Addr,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := rdb.Ping(ctx).Err()
	if err != nil {
		panic(err)
	}
	return &Redis{Client: rdb}
}

func (r *Redis) Wrap(ctx context.Context, f func(ctx context.Context) (any, string, error)) (any, error) {
	ctx, span := mtrace.StartSpan(ctx, "redis", trace.WithSpanKind(trace.SpanKindInternal))
	defer mtrace.EndSpan(span)
	ret, cmd, error := f(ctx)
	span.SetAttributes(mtrace.RedisExecCmd.String(cmd))
	if error != nil {
		span.SetAttributes(mtrace.RedisExecError.String(error.Error()))
	}
	return ret, error
}
