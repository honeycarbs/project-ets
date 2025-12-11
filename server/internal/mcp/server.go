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
	"github.com/honeycarbs/project-ets/internal/domain/job"
	"github.com/honeycarbs/project-ets/internal/mcp/tools"
	"github.com/honeycarbs/project-ets/pkg/logging"
	n4j "github.com/honeycarbs/project-ets/pkg/neo4j"
)

// Server wraps the MCP SDK with an HTTP listener
type Server struct {
	logger *logging.Logger
	config config.Config

	srv         *http.Server
	started     atomic.Bool
	neo4jClient *n4j.Client
}

// Option allows callers to customize server resources
type Option func(*Resources)

// WithJobService injects the job service used by job_search
func WithJobService(service job.Service) Option {
	return func(res *Resources) {
		if service != nil {
			res.JobService = service
		}
	}
}

// WithKeywordRepository injects the keyword repository used by persist_keywords
func WithKeywordRepository(repo tools.KeywordRepository) Option {
	return func(res *Resources) {
		if repo != nil {
			res.KeywordRepo = repo
		}
	}
}

// WithAnalysisService injects the analysis service used by job_analysis
func WithAnalysisService(service tools.AnalysisService) Option {
	return func(res *Resources) {
		if service != nil {
			res.AnalysisSvc = service
		}
	}
}

// WithSheetsClient injects the sheets client used by sheets_export
func WithSheetsClient(client tools.SheetsClient) Option {
	return func(res *Resources) {
		if client != nil {
			res.SheetsClient = client
		}
	}
}

// NewServer builds the MCP HTTP server
func NewServer(log *logging.Logger, cfg config.Config, opts ...Option) (*Server, error) {
	impl := &sdkmcp.Implementation{
		Name:    "project-ets",
		Version: "0.1.0",
	}

	mcpServer := sdkmcp.NewServer(impl, nil)

	ctx := context.Background()
	res, err := initializeResources(ctx, cfg, log)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		if opt != nil {
			opt(res)
		}
	}

	registry := NewToolRegistry(log)
	if err := registry.RegisterAll(mcpServer, *res); err != nil {
		return nil, err
	}

	handler := sdkmcp.NewStreamableHTTPHandler(func(req *http.Request) *sdkmcp.Server {
		return mcpServer
	}, nil)

	// Wrap handler with CORS support and SSE headers for Cloud Run
	corsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept")
		
		// Disable buffering for SSE (critical for Cloud Run)
		w.Header().Set("X-Accel-Buffering", "no")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Connection", "keep-alive")
		
		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		// Log connection attempt for debugging
		log.Info("MCP stream connection attempt",
			"method", r.Method,
			"path", r.URL.Path,
			"content-type", r.Header.Get("Content-Type"),
			"accept", r.Header.Get("Accept"),
			"user-agent", r.Header.Get("User-Agent"))
		
		handler.ServeHTTP(w, r)
	})

	mux := http.NewServeMux()
	mux.Handle("/mcp/stream", corsHandler)
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
		logger:      log,
		config:      cfg,
		srv:         httpSrv,
		neo4jClient: res.Neo4jClient,
	}, nil
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

	if s.neo4jClient != nil {
		if err := s.neo4jClient.Close(ctx); err != nil {
			s.logger.Warn("error during Neo4j cleanup", "err", err)
		}
	}

	if err := s.srv.Shutdown(ctx); err != nil {
		s.logger.Warn("MCP HTTP server shutdown with error", "err", err)
		return err
	}

	s.logger.Info("MCP HTTP server shutdown complete")
	return nil
}
