package server

import (
	"go-im/api/message"
	"go-im/api/user"
	"go-im/internal/common/middleware/mgrpc"
	"go-im/internal/gateway/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Server struct {
	UserRpc    user.UserClient
	MessageRpc message.MessageClient
}

func NewServer(c *config.Config) *Server {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	if c.Trace.Enable {
		opts = append(opts, grpc.WithChainUnaryInterceptor(mgrpc.UnaryClientTrace()))
		opts = append(opts, grpc.WithChainStreamInterceptor(mgrpc.StreamClientTrace()))
	}
	userAddr := c.UserClient.ParseAddr()
	if userAddr == "" {
		panic("user service address is empty")
	}
	messageAddr := c.MessageClient.ParseAddr()
	if messageAddr == "" {
		panic("message service address is empty")
	}
	userConn, err := grpc.NewClient(userAddr, opts...)
	if err != nil {
		panic(err)
	}
	messageConn, err := grpc.NewClient(messageAddr, opts...)
	if err != nil {
		panic(err)
	}
	return &Server{
		UserRpc:    user.NewUserClient(userConn),
		MessageRpc: message.NewMessageClient(messageConn),
	}
}
