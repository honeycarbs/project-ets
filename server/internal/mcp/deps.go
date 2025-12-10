package mcp

import (
	"context"

	"github.com/honeycarbs/project-ets/internal/config"
	"github.com/honeycarbs/project-ets/internal/domain/job"
	"github.com/honeycarbs/project-ets/internal/mcp/tools"
	"github.com/honeycarbs/project-ets/pkg/logging"
	n4j "github.com/honeycarbs/project-ets/pkg/neo4j"
)

type toolDeps struct {
	jobService  job.Service
	analysisSvc tools.AnalysisService
	
	keywordRepo tools.KeywordRepository

	sheetsClient tools.SheetsClient
	neo4jClient  *n4j.Client // stored for cleanup
}

func defaultToolDeps(cfg config.Config, logger *logging.Logger) toolDeps {
	deps, err := InitializeToolDeps(cfg)
	if err != nil {
		logger.Warn("failed to initialize tool dependencies", "err", err)
		// Return minimal deps with stubs
		return toolDeps{
			keywordRepo:  stubKeywordRepository{},
			analysisSvc:  stubAnalysisService{},
			sheetsClient: stubSheetsClient{},
		}
	}

	logger.Info("Adzuna provider initialized", "country", cfg.Adzuna.Country)
	if deps.neo4jClient != nil {
		logger.Info("Neo4j client initialized", "uri", cfg.Neo4j.URI)
	}

	return *deps
}

// cleanup closes resources that need explicit cleanup
func (d *toolDeps) cleanup(ctx context.Context) error {
	if d.neo4jClient != nil {
		return d.neo4jClient.Close(ctx)
	}
	return nil
}

type stubKeywordRepository struct{}

func (stubKeywordRepository) PersistKeywords(ctx context.Context, records []tools.KeywordRecord) error {
	_ = ctx
	_ = records
	return nil
}

type stubAnalysisService struct{}

func (stubAnalysisService) Analyze(ctx context.Context, params tools.JobAnalysisParams) (tools.JobAnalysisResult, error) {
	_ = ctx
	_ = params
	return tools.JobAnalysisResult{}, nil
}

type stubSheetsClient struct{}

func (stubSheetsClient) Export(ctx context.Context, params tools.SheetsExportParams) (tools.SheetsExportResult, error) {
	_ = ctx
	_ = params
	return tools.SheetsExportResult{}, nil
}
