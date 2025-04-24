package seqserver

import (
	"context"
	"go-im/internal/pkg/redis"
)

type SeqServer interface {
	Get(context.Context, string) (int64, error)
}

type redisSeqServer struct {
	r *redis.Redis
}

func NewRedisSeqServer(r *redis.Redis) SeqServer {
	return &redisSeqServer{r: r}
}

func (rs *redisSeqServer) Get(ctx context.Context, key string) (int64, error) {
	ret, err := rs.r.Wrap(ctx, func(ctx context.Context) (any, string, error) {
		cmd := rs.r.Incr(ctx, key)
		return cmd.Val(), cmd.String(), cmd.Err()
	})
	return ret.(int64), err
}
