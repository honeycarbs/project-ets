package tools

import (
	"context"
	"fmt"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/honeycarbs/project-ets/pkg/logging"
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
	logger  *logging.Logger
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

func RegisterAnalysisTools(server *sdkmcp.Server, repo KeywordRepository, svc AnalysisService, logger *logging.Logger) error {
	handler := jobAnalysisTool{service: svc, logger: logger}
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "job_analysis",
		Description: "Summarize stored job graphs against a candidate profile using Graph RAG",
	}, handler.handle)

	persistHandler := persistKeywordsTool{repo: repo, logger: logger}
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "persist_keywords",
		Description: "Store agent-extracted keywords against existing job nodes",
	}, persistHandler.handle)

	if logger != nil {
		logger.Info("analysis tools registered", "tools", []string{"job_analysis", "persist_keywords"})
	}
	return nil
}

func (t jobAnalysisTool) handle(ctx context.Context, req *sdkmcp.CallToolRequest, params *JobAnalysisParams) (*sdkmcp.CallToolResult, any, error) {
	if t.logger != nil {
		t.logger.Debug("job_analysis called")
	}

	if params == nil {
		params = &JobAnalysisParams{}
		if t.logger != nil {
			t.logger.Debug("job_analysis: params is nil, using empty params")
		}
	}

	if t.logger != nil {
		t.logger.Info("job_analysis request",
			"job_ids_count", len(params.JobIDs),
			"job_ids", params.JobIDs,
			"has_profile", params.Profile != "",
			"focus", params.Focus,
		)
	}

	if t.service == nil {
		err := fmt.Errorf("analysis service not configured")
		if t.logger != nil {
			t.logger.Error("job_analysis: service not available", "err", err)
		}
		return nil, nil, err
	}

	result, err := t.service.Analyze(ctx, *params)
	if err != nil {
		if t.logger != nil {
			t.logger.Error("job_analysis: analysis failed",
				"err", err,
				"job_ids", params.JobIDs,
			)
		}
		return nil, nil, fmt.Errorf("analysis failed: %w", err)
	}

	if t.logger != nil {
		t.logger.Info("job_analysis completed successfully",
			"jobs_analyzed", len(result.Jobs),
			"generated_at", result.GeneratedAt,
			"has_notes", result.Notes != "",
		)
		for i, job := range result.Jobs {
			t.logger.Debug("job_analysis result",
				"index", i,
				"job_id", job.JobID,
				"keywords_count", len(job.RecommendedKeywords),
				"has_summary", job.Summary != "",
			)
		}
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
