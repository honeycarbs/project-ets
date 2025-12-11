//go:build wireinject
// +build wireinject

package mcp

import (
	"context"
	"fmt"

	"github.com/google/wire"

	"github.com/honeycarbs/project-ets/internal/config"
	"github.com/honeycarbs/project-ets/internal/domain/analysis"
	"github.com/honeycarbs/project-ets/internal/domain/job"
	adzunaProvider "github.com/honeycarbs/project-ets/internal/domain/job/providers/adzuna"
	"github.com/honeycarbs/project-ets/internal/mcp/tools"
	"github.com/honeycarbs/project-ets/internal/repository"
	storage "github.com/honeycarbs/project-ets/internal/storage/neo4j"
	"github.com/honeycarbs/project-ets/pkg/adzuna"
	n4j "github.com/honeycarbs/project-ets/pkg/neo4j"
	sheetsclient "github.com/honeycarbs/project-ets/pkg/sheets"
)

// InitializeResources creates Resources with all resources wired up
func InitializeResources(ctx context.Context, cfg config.Config) (*Resources, error) {
	wire.Build(
		// Infrastructure - Neo4j
		provideNeo4jConfig,
		n4j.NewClient,

		// Infrastructure - Adzuna
		provideAdzunaConfig,
		adzuna.NewClient,

		// Repositories
		storage.NewJobRepository,
		wire.Bind(new(job.Repository), new(*storage.JobRepository)),
		wire.Bind(new(repository.JobRepository), new(*storage.JobRepository)),
		storage.NewKeywordRepository,
		wire.Bind(new(tools.KeywordRepository), new(*storage.KeywordRepository)),
		storage.NewAnalysisRepository,
		wire.Bind(new(repository.AnalysisRepository), new(*storage.AnalysisRepository)),

		// Providers
		provideAdzunaProvider,
		provideJobProviders,

		// Services
		job.NewServiceWithDeps,
		analysis.NewService,
		wire.Bind(new(tools.AnalysisService), new(*analysis.Service)),

		// Tool resources
		provideSheetsConfig,
		provideSheetsClient,
		provideSheetsClientAdapter,
		newResources,
	)

	return &Resources{}, nil
}

// provideNeo4jConfig extracts Neo4j config from main config
func provideNeo4jConfig(cfg config.Config) n4j.Config {
	return n4j.Config{
		URI:      cfg.Neo4j.URI,
		Username: cfg.Neo4j.Username,
		Password: cfg.Neo4j.Password,
	}
}

// provideAdzunaConfig extracts Adzuna config from main config
func provideAdzunaConfig(cfg config.Config) adzuna.Config {
	return adzuna.Config{
		AppID:   cfg.Adzuna.AppID,
		AppKey:  cfg.Adzuna.AppKey,
		Country: cfg.Adzuna.Country,
	}
}

// provideAdzunaProvider creates an Adzuna provider from client
func provideAdzunaProvider(client *adzuna.Client) (*adzunaProvider.Provider, error) {
	return adzunaProvider.NewProvider(client)
}

// provideJobProviders creates a slice of job providers
func provideJobProviders(adzunaProvider *adzunaProvider.Provider) []job.Provider {
	return []job.Provider{adzunaProvider}
}

// provideSheetsConfig extracts Sheets config from main config
func provideSheetsConfig(cfg config.Config) sheetsclient.Config {
	return sheetsclient.Config{
		CredentialsPath: cfg.Sheets.CredentialsPath,
	}
}

// provideSheetsClient creates a Sheets client
func provideSheetsClient(ctx context.Context, cfg sheetsclient.Config) (*sheetsclient.Client, error) {
	if cfg.CredentialsPath == "" {
		return nil, fmt.Errorf("sheets: GOOGLE_SHEETS_CREDENTIALS_PATH is not set")
	}
	return sheetsclient.NewClient(ctx, cfg)
}

// provideSheetsClientAdapter creates a SheetsClient adapter
func provideSheetsClientAdapter(client *sheetsclient.Client) tools.SheetsClient {
	return &sheetsClientAdapter{client: client}
}

// newResources creates Resources struct
func newResources(
	jobService job.Service,
	jobRepo repository.JobRepository,
	keywordRepo tools.KeywordRepository,
	analysisSvc tools.AnalysisService,
	sheetsClient tools.SheetsClient,
	neo4jClient *n4j.Client,
) *Resources {
	return &Resources{
		JobService:   jobService,
		JobRepo:      jobRepo,
		KeywordRepo:  keywordRepo,
		AnalysisSvc:  analysisSvc,
		SheetsClient: sheetsClient,
		Neo4jClient:  neo4jClient,
	}
}

