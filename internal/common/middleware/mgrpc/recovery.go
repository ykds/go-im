package mgrpc

import (
	"context"
	"go-im/internal/pkg/log"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		var err error
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("panic: %v, stack: %v", r, string(debug.Stack()))
				err = status.Error(codes.Internal, "panic")
			}
		}()
		err = handler(srv, ss)
		return err
	}
}
