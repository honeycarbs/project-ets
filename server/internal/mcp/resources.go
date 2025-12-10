package mcp

import (
	"context"

	"github.com/honeycarbs/project-ets/internal/config"
	"github.com/honeycarbs/project-ets/internal/mcp/tools"
	"github.com/honeycarbs/project-ets/pkg/logging"
)

func initializeResources(cfg config.Config, logger *logging.Logger) (*Resources, error) {
	res, err := InitializeResources(cfg)
	if err != nil {
		logger.Warn("failed to initialize resources", "err", err)
		return &Resources{
			KeywordRepo:  stubKeywordRepository{},
			AnalysisSvc:  stubAnalysisService{},
			SheetsClient: stubSheetsClient{},
		}, err
	}

	logger.Info("Adzuna provider initialized", "country", cfg.Adzuna.Country)
	if res.Neo4jClient != nil {
		logger.Info("Neo4j client initialized", "uri", cfg.Neo4j.URI)
	}

	return res, nil
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
