package mcp

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

// JobAnalysisParams defines the arguments for the job_analysis tool
type JobAnalysisParams struct {
	JobIDs  []string `json:"job_ids,omitempty" jsonschema:"Existing job identifiers stored in Neo4j"`
	Profile string   `json:"profile,omitempty" jsonschema:"Free-form resume/profile to compare"`
	Focus   string   `json:"focus,omitempty" jsonschema:"Optional analysis instruction"`
}

// GraphToolParams defines the arguments for the graph_tool tool
type GraphToolParams struct {
	Cypher  string                 `json:"cypher,omitempty" jsonschema:"Custom Cypher query to run"`
	JobID   string                 `json:"job_id,omitempty"`
	UserID  string                 `json:"user_id,omitempty"`
	Filters map[string]interface{} `json:"filters,omitempty" jsonschema:"Optional label/relation filters"`
}

// SheetsExportParams defines the arguments for the sheets_export tool
type SheetsExportParams struct {
	JobIDs []string          `json:"job_ids,omitempty"`
	Filter map[string]string `json:"filter,omitempty"`
	Sheet  struct {
		SpreadsheetID string `json:"spreadsheet_id" jsonschema:"Google Sheets document ID"`
		Tab           string `json:"tab,omitempty" jsonschema:"Tab name to upsert data"`
	} `json:"sheet" jsonschema:"Destination sheet information"`
}

// registerTools wires all baseline tools into the MCP server as stubs
func registerTools(s *sdkmcp.Server) {
	// job_search
	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "job_search",
		Description: "Search external job boards/APIs, normalize, and store job postings",
	}, jobSearch)

	// job_analysis
	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "job_analysis",
		Description: "Summarize stored job graphs against a candidate profile using Graph RAG",
	}, jobAnalysis)

	// graph_tool
	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "graph_tool",
		Description: "Developer tool for inspecting and debugging the Neo4j knowledge graph",
	}, graphTool)

	// sheets_export
	sdkmcp.AddTool(s, &sdkmcp.Tool{
		Name:        "sheets_export",
		Description: "Export job selections to Google Sheets via the sheets_client integrations",
	}, sheetsExport)
}

func jobSearch(ctx context.Context, req *sdkmcp.CallToolRequest, params *JobSearchParams) (*sdkmcp.CallToolResult, any, error) {
	_ = ctx
	_ = req

	msg := fmt.Sprintf("[job_search] Stub implementation: query=%q location=%q skills=%v", params.Query, params.Location, params.Skills)
	return textResult(msg), nil, nil
}

func jobAnalysis(ctx context.Context, req *sdkmcp.CallToolRequest, params *JobAnalysisParams) (*sdkmcp.CallToolResult, any, error) {
	_ = ctx
	_ = req

	msg := fmt.Sprintf("[job_analysis] Stub implementation: job_ids=%v focus=%q", params.JobIDs, params.Focus)
	return textResult(msg), nil, nil
}

func graphTool(ctx context.Context, req *sdkmcp.CallToolRequest, params *GraphToolParams) (*sdkmcp.CallToolResult, any, error) {
	_ = ctx
	_ = req

	msg := fmt.Sprintf("[graph_tool] Stub implementation: cypher=%q job_id=%q user_id=%q", params.Cypher, params.JobID, params.UserID)
	return textResult(msg), nil, nil
}

func sheetsExport(ctx context.Context, req *sdkmcp.CallToolRequest, params *SheetsExportParams) (*sdkmcp.CallToolResult, any, error) {
	_ = ctx
	_ = req

	msg := fmt.Sprintf("[sheets_export] Stub implementation: job_ids=%v spreadsheet_id=%q tab=%q", params.JobIDs, params.Sheet.SpreadsheetID, params.Sheet.Tab)
	return textResult(msg), nil, nil
}

// Produce a text-only ToolResult
func textResult(msg string) *sdkmcp.CallToolResult {
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{
			&sdkmcp.TextContent{Text: msg},
		},
	}
}
