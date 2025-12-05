package tools

import (
	"context"
	"fmt"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// AnalysisService encapsulates graph/keyword reasoning logic
type AnalysisService interface {
	Analyze(ctx context.Context, params JobAnalysisParams) (JobAnalysisResult, error)
}

// JobAnalysisParams defines the arguments for the job_analysis tool
type JobAnalysisParams struct {
	JobIDs           []string                 `json:"job_ids,omitempty" jsonschema:"Existing job identifiers stored in Neo4j"`
	Profile          string                   `json:"profile,omitempty" jsonschema:"Free-form resume/profile to compare"`
	Focus            string                   `json:"focus,omitempty" jsonschema:"Optional analysis instruction"`
	KeywordOverrides []KeywordOverridePayload `json:"keyword_overrides,omitempty" jsonschema:"Optional keywords supplied by the agent for temporary analysis"`
}

// KeywordOverridePayload allows callers to provide adhoc keyword lists per job
type KeywordOverridePayload struct {
	JobID    string         `json:"job_id" jsonschema:"Job identifier"`
	Keywords []KeywordEntry `json:"keywords" jsonschema:"Keyword list overriding stored values"`
	Notes    string         `json:"notes,omitempty" jsonschema:"Optional annotation for the override"`
}

// JobAnalysisSummary captures per-job analysis output
type JobAnalysisSummary struct {
	JobID          string         `json:"job_id" jsonschema:"Analyzed job identifier"`
	MatchScore     float64        `json:"match_score,omitempty" jsonschema:"Estimated fit score between 0-1"`
	Highlights     []string       `json:"highlights,omitempty" jsonschema:"Reasons this job matches the profile"`
	Gaps           []string       `json:"gaps,omitempty" jsonschema:"Missing or weak skills"`
	Recommended    []KeywordEntry `json:"recommended_keywords,omitempty" jsonschema:"Keywords to emphasize"`
	Summary        string         `json:"summary,omitempty" jsonschema:"Natural language digest of findings"`
	NextActions    []string       `json:"next_actions,omitempty" jsonschema:"Suggested follow-up tasks"`
	SupportingData map[string]any `json:"supporting_data,omitempty" jsonschema:"Optional structured analysis artifacts"`
}

// JobAnalysisResult is the structured response of job_analysis
type JobAnalysisResult struct {
	Jobs        []JobAnalysisSummary `json:"jobs" jsonschema:"Per-job analysis results"`
	GeneratedAt time.Time            `json:"generated_at" jsonschema:"Timestamp when analysis completed"`
	Notes       string               `json:"notes,omitempty" jsonschema:"Global summary or caveats"`
}

type jobAnalysisTool struct {
	service AnalysisService
}

// WithJobAnalysis registers the job_analysis tool
func WithJobAnalysis(service AnalysisService) Option {
	return func(reg *registry) {
		handler := jobAnalysisTool{service: service}
		sdkmcp.AddTool(reg.server, &sdkmcp.Tool{
			Name:        "job_analysis",
			Description: "Summarize stored job graphs against a candidate profile using Graph RAG",
		}, handler.handle)
	}
}

func (t jobAnalysisTool) handle(ctx context.Context, req *sdkmcp.CallToolRequest, params *JobAnalysisParams) (*sdkmcp.CallToolResult, any, error) {
	_ = ctx
	_ = req
	_ = t.service

	var jobIDs []string
	focus := ""
	if params != nil {
		jobIDs = params.JobIDs
		focus = params.Focus
	}

	now := time.Now().UTC()
	result := JobAnalysisResult{
		GeneratedAt: now,
	}

	msg := fmt.Sprintf("[job_analysis] Stub implementation: received %d job id(s) with focus=%q", len(jobIDs), focus)
	return textResult(msg), result, nil
}
