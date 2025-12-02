package mcp

import (
	"context"
	"errors"
	"sync"

	"github.com/honeycarbs/project-ets/internal/config"
	"github.com/honeycarbs/project-ets/pkg/logging"
)

type Server struct {
	log    *logging.Logger
	config config.Config

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func NewServer(log *logging.Logger, cfg config.Config) *Server {
	return &Server{
		log:    log,
		config: cfg,
	}
}

func (s *Server) Run() error {
	s.mu.Lock()
	if s.done != nil {
		s.mu.Unlock()
		return errors.New("mcp server is already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	s.cancel = cancel
	s.done = done
	s.mu.Unlock()

	s.log.Info("MCP server starting (stub)")

	// TODO: replace with real MCP stdio loop that respects ctx.Done().
	<-ctx.Done()

	s.log.Info("MCP server shutting down", "reason", ctx.Err())

	close(done)

	s.mu.Lock()
	s.cancel = nil
	s.done = nil
	s.mu.Unlock()

	if errors.Is(ctx.Err(), context.Canceled) {
		return nil
	}

	return ctx.Err()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	cancel := s.cancel
	done := s.done
	s.mu.Unlock()

	if cancel == nil || done == nil {
		s.log.Info("shutdown requested but MCP server is not running")
		return nil
	}

	cancel()

	select {
	case <-done:
		s.log.Info("MCP server shutdown complete")
		return nil
	case <-ctx.Done():
		s.log.Warn("timed out waiting for MCP server shutdown", "err", ctx.Err())
		return ctx.Err()
	}
}
