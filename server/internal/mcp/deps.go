package mcp

import (
	"context"

	"github.com/honeycarbs/project-ets/internal/domain"
	"github.com/honeycarbs/project-ets/internal/domain/job"
	"github.com/honeycarbs/project-ets/internal/mcp/tools"
)

type toolDeps struct {
	jobService   job.Service
	keywordRepo  tools.KeywordRepository
	analysisSvc  tools.AnalysisService
	sheetsClient tools.SheetsClient
}

func defaultToolDeps() toolDeps {
	return toolDeps{
		jobService:   stubJobService{},
		keywordRepo:  stubKeywordRepository{},
		analysisSvc:  stubAnalysisService{},
		sheetsClient: stubSheetsClient{},
	}
}

type stubJobService struct{}

func (stubJobService) Search(ctx context.Context, query string, filters domain.JobSearchFilters) (domain.JobSearchResult, error) {
	_ = ctx
	_ = query
	_ = filters
	return domain.JobSearchResult{}, nil
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
