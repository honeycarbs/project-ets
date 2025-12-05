package tools

import (
	"context"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// KeywordEntry represents a single extracted keyword
type KeywordEntry struct {
	Value      string   `json:"value" jsonschema:"Keyword text"`
	Confidence *float64 `json:"confidence,omitempty" jsonschema:"Optional confidence between 0-1"`
	Notes      string   `json:"notes,omitempty" jsonschema:"Free-form annotation from the agent"`
}

// KeywordRecord captures the keyword set for a given job
type KeywordRecord struct {
	JobID    string         `json:"job_id" jsonschema:"Canonical job identifier to tag"`
	Keywords []KeywordEntry `json:"keywords" jsonschema:"Extracted keyword list"`
	Source   string         `json:"source,omitempty" jsonschema:"Optional agent/run label"`
}

// KeywordRepository persists keyword records downstream
type KeywordRepository interface {
	PersistKeywords(ctx context.Context, records []KeywordRecord) error
}

// PersistKeywordsParams defines the arguments for the persist_keywords tool
type PersistKeywordsParams struct {
	Records []KeywordRecord `json:"records" jsonschema:"Keyword payloads to persist"`
}

// PersistKeywordsResult represents a summary of the persist operation
type PersistKeywordsResult struct {
	JobIDs       []string `json:"job_ids" jsonschema:"Job identifiers that were processed"`
	SavedRecords int      `json:"saved_records" jsonschema:"Number of keyword records persisted"`
	Message      string   `json:"message,omitempty" jsonschema:"Optional status message"`
}

type persistKeywordsTool struct {
	repo KeywordRepository
}

// WithPersistKeywords registers the persist_keywords tool
func WithPersistKeywords(repo KeywordRepository) Option {
	return func(reg *registry) {
		handler := persistKeywordsTool{repo: repo}
		sdkmcp.AddTool(reg.server, &sdkmcp.Tool{
			Name:        "persist_keywords",
			Description: "Store agent-extracted keywords against existing job nodes",
		}, handler.handle)
	}
}

func (t persistKeywordsTool) handle(ctx context.Context, req *sdkmcp.CallToolRequest, params *PersistKeywordsParams) (*sdkmcp.CallToolResult, any, error) {
	_ = ctx
	_ = req
	_ = t.repo

	result := PersistKeywordsResult{}
	if params != nil {
		result.SavedRecords = len(params.Records)
		result.JobIDs = make([]string, 0, len(params.Records))
		for _, record := range params.Records {
			if record.JobID != "" {
				result.JobIDs = append(result.JobIDs, record.JobID)
			}
		}
	}

	if result.Message == "" {
		result.Message = "keywords accepted (stub)"
	}

	msg := fmt.Sprintf("[persist_keywords] Stub implementation: received %d record(s)", result.SavedRecords)
	return textResult(msg), result, nil
}
