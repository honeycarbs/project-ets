package tools

import (
	"context"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// JobSearchParams defines the arguments for the job_search tool
type JobSearchParams struct {
	Query    string   `json:"query" jsonschema:"Natural language job search query"`
	Location string   `json:"location,omitempty" jsonschema:"Preferred location filter"`
	Remote   *bool    `json:"remote,omitempty" jsonschema:"Whether to restrict to remote postings"`
	Skills   []string `json:"skills,omitempty" jsonschema:"List of required skills"`
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

	msg := fmt.Sprintf("[job_search] Stub implementation: query=%q location=%q skills=%v", params.Query, params.Location, params.Skills)
	return textResult(msg), nil, nil
}
