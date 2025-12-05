package tools

import (
	"context"
	"fmt"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
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

// WithJobSearch registers the job_search tool
func WithJobSearch() Option {
	return func(reg *registry) {
		sdkmcp.AddTool(reg.server, &sdkmcp.Tool{
			Name:        "job_search",
			Description: "Search external job boards/APIs, normalize, and store job postings",
		}, jobSearch)
	}
}

func jobSearch(ctx context.Context, req *sdkmcp.CallToolRequest, params *JobSearchParams) (*sdkmcp.CallToolResult, any, error) {
	_ = ctx
	_ = req

	query := ""
	location := ""
	var skills []string
	if params != nil {
		query = params.Query
		location = params.Location
		skills = params.Skills
	}

	now := time.Now().UTC()
	result := JobSearchResult{
		Jobs: []JobSearchJob{
			{},
		},
		FetchedAt:   now,
		SourceCount: 0,
	}

	msg := fmt.Sprintf("[job_search] Stub implementation: returning %d empty job for query=%q location=%q skills=%v", len(result.Jobs), query, location, skills)
	return textResult(msg), result, nil
}
