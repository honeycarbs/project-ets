package job

import (
	"context"

	"github.com/honeycarbs/project-ets/internal/domain"
)

// Provider represents an external job data source (LinkedIn, Indeed, mock API, etc.)
type Provider interface {
	// e.g. "linkedin" or "Indeed"
	Name() string

	// Search returns normalized jobs for a query
	Search(ctx context.Context, query string, filters domain.JobSearchFilters) ([]domain.Job, error)
}
