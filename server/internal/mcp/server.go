package mcp

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/honeycarbs/project-ets/internal/config"
	"github.com/honeycarbs/project-ets/internal/mcp/tools"
	"github.com/honeycarbs/project-ets/pkg/logging"
)

// Server wraps the MCP SDK with an HTTP listener
type Server struct {
	logger *logging.Logger
	config config.Config

	srv     *http.Server
	started atomic.Bool
}

// NewServer builds the MCP HTTP server
func NewServer(log *logging.Logger, cfg config.Config) *Server {
	impl := &sdkmcp.Implementation{
		Name:    "project-ets",
		Version: "0.1.0",
	}

	mcpServer := sdkmcp.NewServer(impl, nil)

	// Register stub tools
	tools.Register(
		mcpServer,
		tools.WithJobSearch(),
		tools.WithPersistKeywords(),
		tools.WithJobAnalysis(),
		tools.WithGraphTool(),
		tools.WithSheetsExport(),
	)

	handler := sdkmcp.NewStreamableHTTPHandler(func(req *http.Request) *sdkmcp.Server {
		return mcpServer
	}, nil)

	mux := http.NewServeMux()
	mux.Handle("/mcp/stream", handler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	httpSrv := &http.Server{
		Addr:              net.JoinHostPort(cfg.Host, cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &Server{
		logger: log,
		config: cfg,
		srv:    httpSrv,
	}
}

// Run starts the HTTP server until shutdown
func (s *Server) Run() error {
	if !s.started.CompareAndSwap(false, true) {
		return nil
	}

	s.logger.Info("MCP HTTP server listening", "addr", s.srv.Addr)

	if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutdown requested for MCP HTTP server")
	if err := s.srv.Shutdown(ctx); err != nil {
		s.logger.Warn("MCP HTTP server shutdown with error", "err", err)
		return err
	}

	s.logger.Info("MCP HTTP server shutdown complete")
	return nil
}
