package tools

import (
	"context"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// JobAnalysisParams defines the arguments for the job_analysis tool
type JobAnalysisParams struct {
	JobIDs  []string `json:"job_ids,omitempty" jsonschema:"Existing job identifiers stored in Neo4j"`
	Profile string   `json:"profile,omitempty" jsonschema:"Free-form resume/profile to compare"`
	Focus   string   `json:"focus,omitempty" jsonschema:"Optional analysis instruction"`
}

// WithJobAnalysis registers the job_analysis tool
func WithJobAnalysis() Option {
	return func(reg *registry) {
		sdkmcp.AddTool(reg.server, &sdkmcp.Tool{
			Name:        "job_analysis",
			Description: "Summarize stored job graphs against a candidate profile using Graph RAG",
		}, jobAnalysis)
	}
}

func jobAnalysis(ctx context.Context, req *sdkmcp.CallToolRequest, params *JobAnalysisParams) (*sdkmcp.CallToolResult, any, error) {
	_ = ctx
	_ = req

	msg := fmt.Sprintf("[job_analysis] Stub implementation: job_ids=%v focus=%q", params.JobIDs, params.Focus)
	return textResult(msg), nil, nil
}
