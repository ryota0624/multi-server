package multiserver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type loopserver struct {
	breakLoopAfter time.Duration
	serverErr      error
}

func (s *loopserver) Start(ctx context.Context) error {
	c := time.After(s.breakLoopAfter)
	<-c
	return s.serverErr
}
func (s *loopserver) GracefullyShutdown(ctx context.Context) error {
	return nil
}

func Test_managedServer_Start_PropageteContextDone(t *testing.T) {
	serverFinish := time.Second * 5
	m := &managedServer{
		inner: &loopserver{
			breakLoopAfter: serverFinish,
		},
		shutdownStatus:   &ShutdownStatus{},
		occurredStartErr: nil,
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-time.After(time.Second * 3)
		cancel()
	}()
	err := m.Start(ctx)

	assert.ErrorIs(t, err, ErrServerStopByContextDone)
}

func Test_managedServer_Start_ReturnUnderServerStartErr(t *testing.T) {
	serverFinish := time.Second * 5
	anyErr := errors.New("anyErr")
	m := &managedServer{
		inner: &loopserver{
			breakLoopAfter: serverFinish,
			serverErr:      anyErr,
		},
		shutdownStatus:   &ShutdownStatus{},
		occurredStartErr: nil,
	}

	err := m.Start(context.Background())

	assert.ErrorIs(t, err, anyErr)
}

func TestServers_Start_ReturnErrWhenOneServerDown(t *testing.T) {
	anyErr := errors.New("anyErr")

	servers := NewServers().
		Resister(
			&loopserver{
				breakLoopAfter: time.Hour,
				serverErr:      nil,
			},
		).
		Resister(
			&loopserver{
				breakLoopAfter: time.Second,
				serverErr:      anyErr,
			},
		)

	ctx := context.Background()
	err := servers.Start(ctx)
	assert.ErrorIs(t, err, anyErr)
}

func TestServers_Register_Compile(t *testing.T) {
	_ = NewServers().
		Resister(
			NewGrpcServer(nil, nil),
		).
		Resister(
			NewHttpServer(nil, nil),
		)
}
