//go:build wireinject
// +build wireinject

package mcp

import (
	"github.com/google/wire"

	"github.com/honeycarbs/project-ets/internal/config"
	"github.com/honeycarbs/project-ets/internal/domain/job"
	adzunaProvider "github.com/honeycarbs/project-ets/internal/domain/job/providers/adzuna"
	storage "github.com/honeycarbs/project-ets/internal/storage/neo4j"
	"github.com/honeycarbs/project-ets/pkg/adzuna"
	n4j "github.com/honeycarbs/project-ets/pkg/neo4j"
)

// InitializeToolDeps creates toolDeps with all dependencies wired up
func InitializeToolDeps(cfg config.Config) (*toolDeps, error) {
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

		// Providers
		provideAdzunaProvider,
		provideJobProviders,

		// Services
		job.NewServiceWithDeps,

		// Tool dependencies - stubs
		provideStubKeywordRepository,
		provideStubAnalysisService,
		provideStubSheetsClient,
		newToolDeps,
	)

	return &toolDeps{}, nil
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

// provideStubKeywordRepository provides stub keyword repository
func provideStubKeywordRepository() stubKeywordRepository {
	return stubKeywordRepository{}
}

// provideStubAnalysisService provides stub analysis service
func provideStubAnalysisService() stubAnalysisService {
	return stubAnalysisService{}
}

// provideStubSheetsClient provides stub sheets client
func provideStubSheetsClient() stubSheetsClient {
	return stubSheetsClient{}
}

// newToolDeps creates toolDeps struct
func newToolDeps(
	jobService job.Service,
	keywordRepo stubKeywordRepository,
	analysisSvc stubAnalysisService,
	sheetsClient stubSheetsClient,
	neo4jClient *n4j.Client,
) *toolDeps {
	return &toolDeps{
		jobService:   jobService,
		keywordRepo:  keywordRepo,
		analysisSvc:  analysisSvc,
		sheetsClient: sheetsClient,
		neo4jClient:  neo4jClient,
	}
}

