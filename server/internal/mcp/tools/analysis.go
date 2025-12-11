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
	JobIDs  []string `json:"job_ids,omitempty" jsonschema:"Job identifiers stored in Neo4j"`
	Profile string   `json:"profile,omitempty" jsonschema:"Free-form resume/profile to compare"`
	Focus   string   `json:"focus,omitempty" jsonschema:"Optional analysis instruction"`
}

// JobAnalysisSummary captures per-job graph context for LLM analysis
type JobAnalysisSummary struct {
	JobID               string         `json:"job_id" jsonschema:"Job identifier"`
	Summary             string         `json:"summary,omitempty" jsonschema:"Job title and company"`
	RecommendedKeywords []KeywordEntry `json:"keywords,omitempty" jsonschema:"Extracted keywords from graph"`
	SupportingData      map[string]any `json:"data,omitempty" jsonschema:"Full job context for LLM analysis"`
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

func RegisterAnalysisTools(server *sdkmcp.Server, repo KeywordRepository, svc AnalysisService) error {
	handler := jobAnalysisTool{service: svc}
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "job_analysis",
		Description: "Summarize stored job graphs against a candidate profile using Graph RAG",
	}, handler.handle)

	persistHandler := persistKeywordsTool{repo: repo}
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "persist_keywords",
		Description: "Store agent-extracted keywords against existing job nodes",
	}, persistHandler.handle)

	return nil
}

func (t jobAnalysisTool) handle(ctx context.Context, _ *sdkmcp.CallToolRequest, params *JobAnalysisParams) (*sdkmcp.CallToolResult, any, error) {
	if params == nil {
		params = &JobAnalysisParams{}
	}

	result, err := t.service.Analyze(ctx, *params)
	if err != nil {
		return nil, nil, fmt.Errorf("analysis failed: %w", err)
	}

	msg := t.formatResponse(result)
	return textResult(msg), result, nil
}

func (t jobAnalysisTool) formatResponse(result JobAnalysisResult) string {
	if len(result.Jobs) == 0 {
		return "[job_analysis] No jobs found for provided IDs"
	}

	msg := fmt.Sprintf("[job_analysis] Retrieved %d job(s) with graph context\n", len(result.Jobs))

	for _, job := range result.Jobs {
		msg += fmt.Sprintf("\n• %s (keywords: %d)", job.Summary, len(job.RecommendedKeywords))
	}

	return msg
}
