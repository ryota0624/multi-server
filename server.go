package multiserver

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

type Server interface {
	Start(ctx context.Context) error
	GracefullyShutdown(ctx context.Context) error
}

type ShutdownStatus struct {
	initiated int32
}

func (s *ShutdownStatus) ShutdownInitiate() {
	atomic.StoreInt32(&(s.initiated), int32(1))
}

func (s *ShutdownStatus) IsShutdownInitiated() bool {
	if atomic.LoadInt32(&(s.initiated)) != 0 {
		return true
	}
	return false
}

type Servers struct {
	servers         []*managedServer
	shutdownStatus  *ShutdownStatus
	shutdownTimeout time.Duration
	shutdownCtx     context.Context
}

func NewServers() *Servers {
	return &Servers{
		servers:         []*managedServer{},
		shutdownStatus:  &ShutdownStatus{},
		shutdownTimeout: 0,
	}
}

func (servers *Servers) ShutdownTimout(t time.Duration) *Servers {
	servers.shutdownTimeout = t
	return servers
}

func (servers *Servers) Resister(server Server) *Servers {
	servers.servers = append(servers.servers, &managedServer{
		inner:            server,
		occurredStartErr: nil,
		shutdownStatus:   &ShutdownStatus{},
	})

	return servers
}

func (servers *Servers) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	eg, ctx := errgroup.WithContext(ctx)
	for _, server := range servers.servers {
		s := server
		eg.Go(func() error {
			err := s.Start(ctx)
			return err
		})
	}

	err := eg.Wait()
	if err != nil {
		cancel()
		return fmt.Errorf("a server was dawn: %w", err)
	}

	return nil
}

func (servers *Servers) WaitShutdown() {
	if servers.shutdownCtx == nil {
		return
	}

	<-servers.shutdownCtx.Done()
	return
}

var (
	ErrShutdownAlreadyInitiated = errors.New("already initiated")
)

func (servers *Servers) GracefullyShutdown() error {
	if servers.shutdownStatus.IsShutdownInitiated() {
		return ErrShutdownAlreadyInitiated
	}
	servers.shutdownStatus.ShutdownInitiate()
	ctx, cancel := context.WithTimeout(context.Background(), servers.shutdownTimeout)
	defer cancel()
	eg := &errgroup.Group{}
	for _, server := range servers.servers {
		s := server
		eg.Go(func() error {
			err := s.GracefullyShutdown(ctx)
			if errors.Is(err, ErrShutdownAlreadyInitiated) || errors.Is(err, ErrServerAlreadyDawn) {
				return nil
			}
			return err
		})
	}

	return eg.Wait()
}

func (servers *Servers) IsShutdownInitiated() bool {
	return servers.shutdownStatus.IsShutdownInitiated()
}

func (servers *Servers) EnableShutdownOnTerminateSignal() *Servers {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt, os.Kill)
		defer stop()
		<-ctx.Done()
		servers.GracefullyShutdown()
		cancel()
	}()

	servers.shutdownCtx = ctx

	return servers
}

type managedServer struct {
	inner            Server
	shutdownStatus   *ShutdownStatus
	occurredStartErr error
}

var (
	ErrServerStopByContextDone = errors.New("server stop by context done")
)

func (m *managedServer) Start(ctx context.Context) error {
	resultChan := make(chan error)
	go func() {
		err := m.inner.Start(ctx)
		if !errors.Is(err, http.ErrServerClosed) {
			m.occurredStartErr = err
			resultChan <- err
		} else {
			resultChan <- nil
		}

		close(resultChan)
	}()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				return fmt.Errorf("%v: %w", ctx.Err(), ErrServerStopByContextDone)
			}
			return nil
		case err := <-resultChan:
			return err
		}
	}
}

var (
	ErrServerAlreadyDawn = errors.New("server already dawn")
)

func (m *managedServer) GracefullyShutdown(ctx context.Context) error {
	if m.occurredStartErr != nil {
		return ErrServerAlreadyDawn
	}

	if m.shutdownStatus.IsShutdownInitiated() {
		return ErrShutdownAlreadyInitiated
	}
	log.Println("GracefullyShutdown initiated")
	return m.inner.GracefullyShutdown(ctx)
}

func (m *managedServer) IsShutdownInitiated() bool {
	return m.shutdownStatus.IsShutdownInitiated()
}
