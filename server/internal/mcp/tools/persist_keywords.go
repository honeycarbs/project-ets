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

// PersistKeywordsParams defines the arguments for the persist_keywords tool
type PersistKeywordsParams struct {
	Records []KeywordRecord `json:"records" jsonschema:"Keyword payloads to persist"`
}

// WithPersistKeywords registers the persist_keywords tool
func WithPersistKeywords() Option {
	return func(reg *registry) {
		sdkmcp.AddTool(reg.server, &sdkmcp.Tool{
			Name:        "persist_keywords",
			Description: "Store agent-extracted keywords against existing job nodes",
		}, persistKeywords)
	}
}

func persistKeywords(ctx context.Context, req *sdkmcp.CallToolRequest, params *PersistKeywordsParams) (*sdkmcp.CallToolResult, any, error) {
	_ = ctx
	_ = req

	count := 0
	if params != nil {
		count = len(params.Records)
	}

	msg := fmt.Sprintf("[persist_keywords] Stub implementation: received %d record(s)", count)
	return textResult(msg), nil, nil
}
