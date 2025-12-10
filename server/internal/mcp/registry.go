package mcp

import (
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/honeycarbs/project-ets/internal/domain/job"
	"github.com/honeycarbs/project-ets/internal/mcp/tools"
	"github.com/honeycarbs/project-ets/pkg/logging"
	n4j "github.com/honeycarbs/project-ets/pkg/neo4j"
)

type ToolRegistry struct {
	logger *logging.Logger
}

type Resources struct {
	JobService   job.Service
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
		return err
	}

	if err := tools.RegisterAnalysisTools(server, res.KeywordRepo, res.AnalysisSvc); err != nil {
		return err
	}

	if err := tools.RegisterExportTools(server, res.SheetsClient); err != nil {
		return err
	}

	if err := tools.RegisterGraphTool(server); err != nil {
		return err
	}

	return nil
}
