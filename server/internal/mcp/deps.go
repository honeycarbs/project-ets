package mcp

import (
	"context"
	"fmt"

	"github.com/honeycarbs/project-ets/internal/config"
	"github.com/honeycarbs/project-ets/internal/domain"
	"github.com/honeycarbs/project-ets/internal/domain/job"
	adzunaProvider "github.com/honeycarbs/project-ets/internal/domain/job/providers/adzuna"
	"github.com/honeycarbs/project-ets/internal/mcp/tools"
	adzuna "github.com/honeycarbs/project-ets/pkg/adzuna"
	"github.com/honeycarbs/project-ets/pkg/logging"
)

type toolDeps struct {
	jobService   job.Service
	keywordRepo  tools.KeywordRepository
	analysisSvc  tools.AnalysisService
	sheetsClient tools.SheetsClient
}

func defaultToolDeps(cfg config.Config, logger *logging.Logger) toolDeps {
	deps := toolDeps{
		keywordRepo:  stubKeywordRepository{},
		analysisSvc:  stubAnalysisService{},
		sheetsClient: stubSheetsClient{},
	}

	if svc, err := buildAdzunaJobService(cfg); err == nil {
		deps.jobService = svc
		logger.Info("Adzuna provider initialized", "country", cfg.Adzuna.Country)
	} else {
		logger.Warn("failed to initialize Adzuna provider", "err", err)
	}

	return deps
}

func buildAdzunaJobService(cfg config.Config) (job.Service, error) {
	if cfg.Adzuna.AppID == "" || cfg.Adzuna.AppKey == "" {
		return nil, fmt.Errorf("adzuna credentials missing")
	}

	client, err := adzuna.NewClient(adzuna.Config{
		AppID:   cfg.Adzuna.AppID,
		AppKey:  cfg.Adzuna.AppKey,
		Country: cfg.Adzuna.Country,
	})
	if err != nil {
		return nil, err
	}

	provider, err := adzunaProvider.NewProvider(client)
	if err != nil {
		return nil, err
	}

	repo := stubJobRepository{}

	return job.NewService(
		job.WithProviders(provider),
		job.WithRepository(repo),
	)
}

type stubJobRepository struct{}

func (stubJobRepository) UpsertJobs(ctx context.Context, jobs []domain.Job) error {
	_ = ctx
	_ = jobs
	return nil
}

func (stubJobRepository) FindByIDs(ctx context.Context, ids []domain.JobID) ([]domain.Job, error) {
	_ = ctx
	_ = ids
	return nil, nil
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
