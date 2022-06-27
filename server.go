package multiserver

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"

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
	servers        []*managedServer
	shutdownStatus *ShutdownStatus
}

func NewServers() *Servers {
	return &Servers{
		servers:        []*managedServer{},
		shutdownStatus: &ShutdownStatus{},
	}
}

func (servers *Servers) Resister(server Server) *Servers {
	servers.servers = append(servers.servers, &managedServer{
		inner:            server,
		occurredStartErr: nil,
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

var (
	ErrShutdownAlreadyInitiated = errors.New("already initiated")
)

func (servers *Servers) GracefullyShutdown(ctx context.Context) error {
	if servers.shutdownStatus.IsShutdownInitiated() {
		return ErrShutdownAlreadyInitiated
	}
	servers.shutdownStatus.ShutdownInitiate()
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
		m.occurredStartErr = err
		resultChan <- err
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
