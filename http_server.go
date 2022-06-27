package multiserver

import (
	"context"
	"net"
	"net/http"
)

type HttpServer struct {
	*http.Server
	lis net.Listener
}

var _ Server = (*HttpServer)(nil)

func (s *HttpServer) Start(_ context.Context) error {
	return s.Serve(s.lis)
}
func (s *HttpServer) GracefullyShutdown(ctx context.Context) error {
	return s.Shutdown(ctx)
}

func NewHttpServer(server *http.Server, lis net.Listener) *HttpServer {
	return &HttpServer{
		Server: server,
		lis:    lis,
	}
}
