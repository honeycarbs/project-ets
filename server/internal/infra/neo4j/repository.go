package neo4j

import (
	"context"

	"github.com/honeycarbs/project-ets/internal/domain"
)

// Repository implements job.Repository with Neo4j
type Repository struct {
	// TODO: add Neo4j driver fields such as neo4j.DriverWithContext
}

// NewRepository creates a Repository
func NewRepository() *Repository {
	return &Repository{}
}

// UpsertJobs will merge and set job data in Neo4j
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
