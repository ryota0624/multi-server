package multiserver

import (
	"context"
	"net"

	"google.golang.org/grpc"
)

type GrpcServer struct {
	*grpc.Server
	lis net.Listener
}

var _ Server = (*HttpServer)(nil)

func (s *GrpcServer) Start(_ context.Context) error {
	return s.Serve(s.lis)
}
func (s *GrpcServer) GracefullyShutdown(_ context.Context) error {
	s.Stop()
	return nil
}

func NewGrpcServer(server *grpc.Server, lis net.Listener) *GrpcServer {
	return &GrpcServer{
		Server: server,
		lis:    lis,
	}
}
