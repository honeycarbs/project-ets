package job

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/honeycarbs/project-ets/internal/domain"
)

type Service interface {
	Search(ctx context.Context, query string, filters domain.JobSearchFilters) (domain.JobSearchResult, error)
}

// Option configures Service
type Option func(*config)

type config struct {
	providers []Provider
	repo      Repository
	clock     func() time.Time
}

// WithProviders sets job providers
func WithProviders(providers ...Provider) Option {
	return func(c *config) {
		c.providers = providers
	}
}

// WithRepository sets the repository
func WithRepository(repo Repository) Option {
	return func(c *config) {
		c.repo = repo
	}
}

// WithClock sets a custom clock
func WithClock(clock func() time.Time) Option {
	return func(c *config) {
		c.clock = clock
	}
}

// NewService builds Service from options
func NewService(opts ...Option) (Service, error) {
	cfg := &config{
		clock: time.Now,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.repo == nil {
		return nil, fmt.Errorf("job.Service: repository is required")
	}
	if len(cfg.providers) == 0 {
		return nil, fmt.Errorf("job.Service: at least one provider is required")
	}

	return &service{
		providers: cfg.providers,
		repo:      cfg.repo,
		clock:     cfg.clock,
	}, nil
}

// NewServiceWithDeps creates a Service with direct dependencies (Wire-compatible)
func NewServiceWithDeps(repo Repository, providers []Provider) (Service, error) {
	if repo == nil {
		return nil, fmt.Errorf("job.Service: repository is required")
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("job.Service: at least one provider is required")
	}

	return &service{
		providers: providers,
		repo:      repo,
		clock:     time.Now,
	}, nil
}

type service struct {
	providers []Provider
	repo      Repository
	clock     func() time.Time
}

// Search queries providers and stores results
func (s *service) Search(
	ctx context.Context,
	query string,
	filters domain.JobSearchFilters,
) (domain.JobSearchResult, error) {
	now := s.clock()

	if query == "" {
		return domain.JobSearchResult{}, fmt.Errorf("query is required")
	}

	type key struct {
		source     string
		externalID string
	}
	dedup := make(map[key]domain.Job)
	sourceCount := 0

	for _, p := range s.providers {
		jobs, err := p.Search(ctx, query, filters)
		if err != nil {
			continue
		}
		if len(jobs) > 0 {
			sourceCount++
		}

		for _, j := range jobs {
			if j.Source == "" || j.ExternalID == "" {
				continue
			}
			k := key{source: j.Source, externalID: j.ExternalID}

			if j.ID == uuid.Nil {
				j.ID = uuid.New()
			}
			if j.FetchedAt.IsZero() {
				j.FetchedAt = now
			}

			dedup[k] = j
		}
	}

	allJobs := make([]domain.Job, 0, len(dedup))
	for _, j := range dedup {
		allJobs = append(allJobs, j)
	}

	if len(allJobs) > 0 {
		if err := s.repo.UpsertJobs(ctx, allJobs); err != nil {
			return domain.JobSearchResult{}, err
		}
	}

	summaries := make([]domain.JobSummary, 0, len(allJobs))
	for _, j := range allJobs {
		summaries = append(summaries, domain.JobSummary{
			ID:       j.ID,
			Title:    j.Title,
			Company:  j.Company.Name,
			Location: j.Location,
			Remote:   j.Remote,
			URL:      j.URL,
			Source:   j.Source,
			Score:    j.Score,
		})
	}

	return domain.JobSearchResult{
		Jobs:        summaries,
		FetchedAt:   now,
		SourceCount: sourceCount,
	}, nil
}
