package repository

import (
	"context"

	"github.com/honeycarbs/project-ets/internal/domain"
)

// JobRepository defines the interface for job storage operations
type JobRepository interface {
	UpsertJobs(ctx context.Context, jobs []domain.Job) error
	FindByIDs(ctx context.Context, ids []domain.JobID) ([]domain.Job, error)
}

