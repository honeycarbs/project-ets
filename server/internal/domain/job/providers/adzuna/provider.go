package adzuna

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/honeycarbs/project-ets/internal/domain"
	jobdomain "github.com/honeycarbs/project-ets/internal/domain/job"
	"github.com/honeycarbs/project-ets/pkg/adzuna"
)

// searchClient describes the subset of the Adzuna client used by the provider.
type searchClient interface {
	SearchJobs(ctx context.Context, query string, params adzuna.SearchParams) ([]adzuna.Job, error)
}

// Provider implements job.Provider using Adzuna API
type Provider struct {
	client searchClient
}

// NewProvider builds an Adzuna provider
func NewProvider(client searchClient) (*Provider, error) {
	if client == nil {
		return nil, fmt.Errorf("adzuna provider: client is required")
	}
	return &Provider{client: client}, nil
}

// Name returns provider identifier
func (p *Provider) Name() string {
	return "adzuna"
}

// Search queries Adzuna and returns normalized jobs
func (p *Provider) Search(ctx context.Context, query string, filters domain.JobSearchFilters) ([]domain.Job, error) {
	if p == nil || p.client == nil {
		return nil, fmt.Errorf("adzuna provider: client is nil")
	}

	params := adzuna.SearchParams{
		Location: filters.Location,
		Remote:   filters.Remote,
		Skills:   filters.Skills,
	}

	respJobs, err := p.client.SearchJobs(ctx, query, params)
	if err != nil {
		return nil, err
	}

	out := make([]domain.Job, 0, len(respJobs))
	for _, j := range respJobs {
		jobID, err := uuid.Parse(j.ID)
		if err != nil {
			jobID = uuid.New()
		}

		out = append(out, domain.Job{
			ID:    jobID,
			Title: j.Title,
			Company: domain.CompanyRef{
				ID:   slugify(j.CompanyName),
				Name: j.CompanyName,
			},
			Location:    j.Location,
			Remote:      j.Remote,
			URL:         j.URL,
			Source:      "adzuna",
			ExternalID:  j.ID,
			Description: j.Description,
			Score:       j.SalaryMax,
			PostedAt:    j.PostedAt,
			FetchedAt:   j.FetchedAt,
		})
	}

	return out, nil
}

var _ jobdomain.Provider = (*Provider)(nil)

func slugify(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.ReplaceAll(s, " ", "-")
}
