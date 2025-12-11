package mcp

import (
	"context"

	"github.com/honeycarbs/project-ets/internal/config"
	"github.com/honeycarbs/project-ets/pkg/logging"
)

func initializeResources(ctx context.Context, cfg config.Config, logger *logging.Logger) (*Resources, error) {
	res, err := InitializeResources(ctx, cfg, logger)
	if err != nil {
		logger.Warn("failed to initialize resources", "err", err)
		return nil, err
	}

	logger.Info("Adzuna provider initialized", "country", cfg.Adzuna.Country)
	if res.Neo4jClient != nil {
		logger.Info("Neo4j client initialized", "uri", cfg.Neo4j.URI)
	}

	return res, nil
}
