package tools

import (
	"context"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// GraphToolParams defines the arguments for the graph_tool tool
type GraphToolParams struct {
	Cypher  string                 `json:"cypher,omitempty" jsonschema:"Custom Cypher query to run"`
	JobID   string                 `json:"job_id,omitempty"`
	UserID  string                 `json:"user_id,omitempty"`
	Filters map[string]interface{} `json:"filters,omitempty" jsonschema:"Optional label/relation filters"`
}

// WithGraphTool registers the graph_tool
func WithGraphTool() Option {
	return func(reg *registry) {
		sdkmcp.AddTool(reg.server, &sdkmcp.Tool{
			Name:        "graph_tool",
			Description: "Developer tool for inspecting and debugging the Neo4j knowledge graph",
		}, graphTool)
	}
}

func graphTool(ctx context.Context, req *sdkmcp.CallToolRequest, params *GraphToolParams) (*sdkmcp.CallToolResult, any, error) {
	_ = ctx
	_ = req

	msg := fmt.Sprintf("[graph_tool] Stub implementation: cypher=%q job_id=%q user_id=%q", params.Cypher, params.JobID, params.UserID)
	return textResult(msg), nil, nil
}
