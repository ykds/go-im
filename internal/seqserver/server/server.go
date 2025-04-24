package server

import (
	"go-im/internal/seqserver/pkg/seqserver"
)

type Server struct {
	SeqServer seqserver.SeqServer
}

func NewServer(seqServer seqserver.SeqServer) *Server {
	return &Server{
		SeqServer: seqServer,
	}
}
