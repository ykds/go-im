package mgrpc

import (
	"context"
	"go-im/internal/pkg/log"
	"runtime/debug"

	"google.golang.org/grpc"
)

func UnaryServerRecovery() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("panic: %v, stack: %v", r, string(debug.Stack()))
			}
		}()
		return handler(ctx, req)
	}
}

func StreamServerRecovery() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("panic: %v, stack: %v", r, string(debug.Stack()))
			}
		}()
		return handler(srv, ss)
	}
}
