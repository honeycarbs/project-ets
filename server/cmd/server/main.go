package main

import (
	"log"
	"net"
	"os"
	"syscall"
	"time"

	"github.com/honeycarbs/project-ets/internal/config"
	"github.com/honeycarbs/project-ets/internal/mcp"
	"github.com/honeycarbs/project-ets/pkg/logging"
	"github.com/honeycarbs/project-ets/pkg/shutdown"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger := logging.New(cfg.LogLevel)
	defer func() { _ = logger.Sync() }()

	srv := mcp.NewServer(logger, cfg)

	go shutdown.Graceful(
		[]os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP},
		srv,
		10*time.Second,
		logger,
	)

	logger.Info("MCP server initialized and starting", "addr", net.JoinHostPort(cfg.Host, cfg.Port))

	if err := srv.Run(); err != nil {
		logger.Error("MCP server exited with error", "err", err)
	} else {
		logger.Info("MCP server stopped")
	}
}
