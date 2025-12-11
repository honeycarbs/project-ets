package mcp

import (
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/honeycarbs/project-ets/internal/domain/job"
	"github.com/honeycarbs/project-ets/internal/mcp/tools"
	"github.com/honeycarbs/project-ets/internal/repository"
	"github.com/honeycarbs/project-ets/pkg/logging"
	n4j "github.com/honeycarbs/project-ets/pkg/neo4j"
)

type ToolRegistry struct {
	logger *logging.Logger
}

type Resources struct {
	JobService   job.Service
	JobRepo      repository.JobRepository
	KeywordRepo  tools.KeywordRepository
	AnalysisSvc  tools.AnalysisService
	SheetsClient tools.SheetsClient
	Neo4jClient  *n4j.Client
}

func NewToolRegistry(logger *logging.Logger) *ToolRegistry {
	return &ToolRegistry{logger: logger}
}

func (r *ToolRegistry) RegisterAll(server *sdkmcp.Server, res Resources) error {
	if err := tools.RegisterJobTools(server, res.JobService, r.logger); err != nil {
		r.logger.Error("failed to register job tools", "err", err)
		return err
	}

	if err := tools.RegisterAnalysisTools(server, res.KeywordRepo, res.AnalysisSvc, r.logger); err != nil {
		r.logger.Error("failed to register analysis tools", "err", err)
		return err
	}

	if err := tools.RegisterExportTools(server, res.SheetsClient, res.JobRepo, r.logger); err != nil {
		r.logger.Error("failed to register export tools", "err", err)
		return err
	}

	if err := tools.RegisterGraphTool(server, res.Neo4jClient, r.logger); err != nil {
		r.logger.Error("failed to register graph tool", "err", err)
		return err
	}

	r.logger.Info("all MCP tools registered successfully")
	return nil
}
