package tools

import (
	"context"
	"errors"
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
	if logger != nil {
		logger.Info("job_search tool registered successfully")
	}
	return nil
}

func (t jobSearchTool) handle(ctx context.Context, req *sdkmcp.CallToolRequest, params *JobSearchParams) (*sdkmcp.CallToolResult, any, error) {
	if t.logger != nil {
		t.logger.Debug("job_search called")
	}

	query := ""
	location := ""
	var skills []string
	var remote *bool
	if params != nil {
		query = params.Query
		location = params.Location
		skills = params.Skills
		remote = params.Remote
	}

	if t.service == nil {
		msg := "job_search unavailable: no job service is configured"
		if t.logger != nil {
			t.logger.Warn("job_search: service not available")
		}
		return textResult(msg), JobSearchResult{}, errors.New(msg)
	}

	if t.logger != nil {
		t.logger.Info("job_search request",
			"query", query,
			"location", location,
			"skills", skills,
			"remote", remote,
		)
	}

	if query == "" {
		err := fmt.Errorf("job_search: query is required")
		if t.logger != nil {
			t.logger.Warn("job_search: missing required query parameter")
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

	if t.logger != nil {
		t.logger.Debug("job_search: calling service.Search",
			"filters", fmt.Sprintf("%+v", filters),
		)
	}

	serviceResult, err := t.service.Search(ctx, query, filters)
	if err != nil {
		if t.logger != nil {
			t.logger.Error("job_search: service failure",
				"err", err,
				"query", query,
				"filters", fmt.Sprintf("%+v", filters),
			)
		}
		return textResult(fmt.Sprintf("job_search failed: %v", err)), JobSearchResult{}, err
	}

	if t.logger != nil {
		t.logger.Debug("job_search: service returned results",
			"jobs_count", len(serviceResult.Jobs),
			"source_count", serviceResult.SourceCount,
			"fetched_at", serviceResult.FetchedAt,
		)
	}

	jobs := make([]JobSearchJob, 0, len(serviceResult.Jobs))
	for i, summary := range serviceResult.Jobs {
		job := JobSearchJob{
			ID:        summary.ID.String(),
			Title:     summary.Title,
			Company:   summary.Company,
			Location:  summary.Location,
			Remote:    summary.Remote,
			URL:       summary.URL,
			Source:    summary.Source,
			Score:     summary.Score,
			FetchedAt: serviceResult.FetchedAt,
		}
		jobs = append(jobs, job)
		
		if t.logger != nil {
			t.logger.Debug("job_search: job processed",
				"index", i,
				"job_id", job.ID,
				"title", job.Title,
				"company", job.Company,
				"source", job.Source,
			)
		}
	}

	result := JobSearchResult{
		Jobs:        jobs,
		FetchedAt:   serviceResult.FetchedAt,
		SourceCount: serviceResult.SourceCount,
	}

	if t.logger != nil {
		t.logger.Info("job_search completed successfully",
			"jobs_count", len(jobs),
			"sources", serviceResult.SourceCount,
			"fetched_at", serviceResult.FetchedAt,
		)
	}

	msg := fmt.Sprintf("[job_search] fetched %d job(s) from %d source(s)\n", len(jobs), serviceResult.SourceCount)
	for _, j := range jobs {
		msg += fmt.Sprintf("  • %s | %s at %s [%s]\n", j.ID, j.Title, j.Company, j.Location)
	}
	return textResult(msg), result, nil
}
