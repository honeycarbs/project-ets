package tools

import (
	"context"
	"fmt"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/honeycarbs/project-ets/internal/domain"
	"github.com/honeycarbs/project-ets/internal/domain/job"
	"github.com/honeycarbs/project-ets/pkg/logging"
)

// JobSearchParams defines the arguments for the job_search tool
type JobSearchParams struct {
	Query    string   `json:"query" jsonschema:"Natural language job search query"`
	Location string   `json:"location,omitempty" jsonschema:"Preferred location filter"`
	Remote   *bool    `json:"remote,omitempty" jsonschema:"Whether to restrict to remote postings"`
	Skills   []string `json:"skills,omitempty" jsonschema:"List of required skills"`
}

// JobSearchJob represents a normalized job returned to the client
type JobSearchJob struct {
	ID          string    `json:"id" jsonschema:"Canonical job identifier"`
	Title       string    `json:"title" jsonschema:"Job title"`
	Company     string    `json:"company" jsonschema:"Company or employer name"`
	Location    string    `json:"location" jsonschema:"Primary listed location"`
	Remote      bool      `json:"remote" jsonschema:"Whether the role is remote-friendly"`
	URL         string    `json:"url,omitempty" jsonschema:"Direct application URL"`
	Source      string    `json:"source,omitempty" jsonschema:"Originating provider e.g. linkedin"`
	Score       float64   `json:"score,omitempty" jsonschema:"Relevance or ranking score"`
	Description string    `json:"description,omitempty" jsonschema:"Canonical job description text"`
	Skills      []string  `json:"skills,omitempty" jsonschema:"Parsed/normalized skill tags"`
	FetchedAt   time.Time `json:"fetched_at" jsonschema:"Timestamp the job was fetched"`
}

// JobSearchResult contains the result payload for job_search
type JobSearchResult struct {
	Jobs        []JobSearchJob `json:"jobs" jsonschema:"Job payloads for downstream tools"`
	FetchedAt   time.Time      `json:"fetched_at" jsonschema:"Search execution timestamp"`
	SourceCount int            `json:"source_count" jsonschema:"How many providers returned results"`
}

type jobSearchTool struct {
	service job.Service
	logger  *logging.Logger
}

// WithJobSearch registers the job_search tool with the provided service
func WithJobSearch(service job.Service, logger *logging.Logger) Option {
	return func(reg *registry) {
		handler := jobSearchTool{
			service: service,
			logger:  logger,
		}
		sdkmcp.AddTool(reg.server, &sdkmcp.Tool{
			Name:        "job_search",
			Description: "Search external job boards/APIs, normalize, and store job postings",
		}, handler.handle)
	}
}

func RegisterJobTools(server *sdkmcp.Server, jobSvc job.Service, logger *logging.Logger) error {
	handler := jobSearchTool{
		service: jobSvc,
		logger:  logger,
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "job_search",
		Description: "Search external job boards/APIs, normalize, and store job postings",
	}, handler.handle)
	return nil
}

func (t jobSearchTool) handle(ctx context.Context, req *sdkmcp.CallToolRequest, params *JobSearchParams) (*sdkmcp.CallToolResult, any, error) {
	query := ""
	location := ""
	var skills []string
	if params != nil {
		query = params.Query
		location = params.Location
		skills = params.Skills
	}

	if t.service == nil {
		msg := "job_search unavailable: no job service is configured"
		if t.logger != nil {
			t.logger.Warn(msg)
		}
		return textResult(msg), JobSearchResult{}, fmt.Errorf(msg)
	}

	if t.logger != nil {
		t.logger.Info("job_search request",
			"query", query,
			"location", location,
			"skills", skills,
		)
	}

	if query == "" {
		err := fmt.Errorf("job_search: query is required")
		if t.logger != nil {
			t.logger.Warn("job_search missing query")
		}
		return textResult("job_search requires a non-empty query"), JobSearchResult{}, err
	}

	filters := domain.JobSearchFilters{
		Location: location,
		Remote:   nil,
		Skills:   skills,
	}
	if params != nil {
		filters.Remote = params.Remote
	}

	serviceResult, err := t.service.Search(ctx, query, filters)
	if err != nil {
		if t.logger != nil {
			t.logger.Error("job_search service failure", "err", err)
		}
		return textResult(fmt.Sprintf("job_search failed: %v", err)), JobSearchResult{}, err
	}

	jobs := make([]JobSearchJob, 0, len(serviceResult.Jobs))
	for _, summary := range serviceResult.Jobs {
		jobs = append(jobs, JobSearchJob{
			ID:        summary.ID.String(),
			Title:     summary.Title,
			Company:   summary.Company,
			Location:  summary.Location,
			Remote:    summary.Remote,
			URL:       summary.URL,
			Source:    summary.Source,
			Score:     summary.Score,
			FetchedAt: serviceResult.FetchedAt,
		})
	}

	result := JobSearchResult{
		Jobs:        jobs,
		FetchedAt:   serviceResult.FetchedAt,
		SourceCount: serviceResult.SourceCount,
	}

	if t.logger != nil {
		t.logger.Info("job_search completed",
			"jobs", len(jobs),
			"sources", serviceResult.SourceCount,
		)
	}

	msg := fmt.Sprintf("[job_search] fetched %d job(s) from %d source(s)\n", len(jobs), serviceResult.SourceCount)
	for _, j := range jobs {
		msg += fmt.Sprintf("  • %s | %s at %s [%s]\n", j.ID, j.Title, j.Company, j.Location)
	}
	return textResult(msg), result, nil
}
