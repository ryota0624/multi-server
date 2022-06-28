package multiserver

import (
	"context"
	"errors"
	"net"

	"google.golang.org/grpc"
)

type GrpcServer struct {
	*grpc.Server
	lis net.Listener
}

var _ Server = (*HttpServer)(nil)

func (s *GrpcServer) Start(_ context.Context) error {
	err := s.Serve(s.lis)
	if errors.Is(err, grpc.ErrServerStopped) {
		return nil
	}
	return err
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
