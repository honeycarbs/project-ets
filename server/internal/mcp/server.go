package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/honeycarbs/project-ets/internal/config"
	"github.com/honeycarbs/project-ets/pkg/logging"
)

var nullID = json.RawMessage("null")

// Server hosts the MCP transport loop over HTTP
type Server struct {
	log     *logging.Logger
	config  config.Config
	router  *Router
	httpSrv *http.Server
	started atomic.Bool
}

// NewServer constructs an MCP server instance
func NewServer(log *logging.Logger, cfg config.Config) *Server {
	s := &Server{
		log:    log,
		config: cfg,
		router: NewRouter(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/mcp/stream", s.handleStream)

	s.httpSrv = &http.Server{
		Addr:              net.JoinHostPort(cfg.Host, cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return s
}

// Run starts processing JSON-RPC requests served over HTTP
func (s *Server) Run() error {
	// Only the first caller can enter the transport loop,
	// if a second goroutine calls Run(), the compare fails,
	// return an error instead of having two loops racing on stdin
	//
	// for now since main calls Run there is no point in this
	// other than safety guard for tests or future improvements
	if !s.started.CompareAndSwap(false, true) {
		return errors.New("mcp server already running")
	}

	s.log.Info("MCP HTTP server listening", "addr", s.httpSrv.Addr)

	if err := s.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Shutdown stops the transport loop and waits for completion
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("shutdown requested for MCP HTTP server")
	if err := s.httpSrv.Shutdown(ctx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			s.log.Warn("timed out waiting for MCP server shutdown", "err", err)
		}
		return err
	}

	s.log.Info("MCP server shutdown complete")
	return nil
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson")

	dec := json.NewDecoder(r.Body)
	defer func() {
		if err := r.Body.Close(); err != nil {
			s.log.Warn("failed to close request body", "err", err)
		}
	}()

	enc := json.NewEncoder(w)

	for {
		var req RPCRequest
		if err := dec.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}

			s.log.Warn("decode error", "err", err)
			_ = enc.Encode(s.errorResponse(nullID, ErrCodeParse, fmt.Sprintf("decode error: %v", err)))
			flusher.Flush()
			return
		}

		if len(req.ID) == 0 {
			s.log.Warn("notification received; ignoring", "method", req.Method)
			continue
		}

		resp := s.handleRequest(r.Context(), req)
		if err := enc.Encode(resp); err != nil {
			s.log.Error("failed to encode MCP response", "err", err)
			return
		}

		flusher.Flush()
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// RegisterTool wires an MCP tool into the router
func (s *Server) RegisterTool(tool Tool) error {
	return s.router.Register(tool)
}

// RegisterTools registers multiple tools
func (s *Server) RegisterTools(tools ...Tool) error {
	for _, tool := range tools {
		if err := s.RegisterTool(tool); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) handleRequest(ctx context.Context, req RPCRequest) RPCResponse {
	if req.JSONRPC != "2.0" {
		return s.errorResponse(req.ID, ErrCodeInvalidRequest, "jsonrpc field must be \"2.0\"")
	}

	switch req.Method {
	case "initialize":
		var params InitializeParams
		if len(req.Params) > 0 {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				return s.errorResponse(req.ID, ErrCodeInvalidParams, fmt.Sprintf("invalid initialize params: %v", err))
			}
		}

		result := InitializeResult{
			ServerInfo: ServerInfo{
				Name:    "project-ets-mcp",
				Version: "0.1.0",
			},
			Capabilities: Capabilities{},
			Tools:        s.router.List(),
		}

		return s.successResponse(req.ID, result)

	case "list_tools":
		return s.successResponse(req.ID, ListToolsResult{
			Tools: s.router.List(),
		})

	case "call_tool":
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return s.errorResponse(req.ID, ErrCodeInvalidParams, fmt.Sprintf("invalid call_tool params: %v", err))
		}

		if params.Name == "" {
			return s.errorResponse(req.ID, ErrCodeInvalidParams, "tool name is required")
		}

		result, err := s.router.Call(ctx, params.Name, params.Params)
		if err != nil {
			return s.errorResponse(req.ID, ErrCodeInternal, err.Error())
		}

		return s.successResponse(req.ID, result)

	default:
		return s.errorResponse(req.ID, ErrCodeMethodNotFound, fmt.Sprintf("method %q is not supported", req.Method))
	}
}

func (s *Server) successResponse(id json.RawMessage, result any) RPCResponse {
	return RPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

func (s *Server) errorResponse(id json.RawMessage, code int, message string) RPCResponse {
	return RPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}
}
