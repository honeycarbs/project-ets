package neo4j

import (
	"context"

	"github.com/honeycarbs/project-ets/internal/domain"
	//"github.com/honeycarbs/project-ets/internal/domain/job"
)

// Repository is a Neo4j-backed implementation of job.Repository
type Repository struct {
	// TODO: add Neo4j driver fields here, e.g. driver neo4j.DriverWithContext
}

// NewRepository constructs a new Repository
func NewRepository() *Repository {
	return &Repository{}
}

// UpsertJobs performs Neo4j MERGE/SET logic when wiring the database (TODO)
func (r *Repository) UpsertJobs(ctx context.Context, jobs []domain.Job) error {
	_ = ctx
	_ = jobs
	return nil
}

func (r *Repository) FindByIDs(ctx context.Context, ids []domain.JobID) ([]domain.Job, error) {
	_ = ctx
	_ = ids
	return nil, nil
}
