package job

import (
	"context"

	"github.com/honeycarbs/project-ets/internal/domain"
)

// Repository persists and loads jobs from storage
type Repository interface {
	// UpsertJobs creates or updates jobs based on Source + ExternalID
	UpsertJobs(ctx context.Context, jobs []domain.Job) error

	// FindByIDs loads full Job records for the given IDs
	FindByIDs(ctx context.Context, ids []domain.JobID) ([]domain.Job, error)
}
