package tools

import (
	"context"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// SheetsExportParams defines the arguments for the sheets_export tool
type SheetsExportParams struct {
	JobIDs []string          `json:"job_ids,omitempty"`
	Filter map[string]string `json:"filter,omitempty"`
	Sheet  struct {
		SpreadsheetID string `json:"spreadsheet_id" jsonschema:"Google Sheets document ID"`
		Tab           string `json:"tab,omitempty" jsonschema:"Tab name to upsert data"`
	} `json:"sheet" jsonschema:"Destination sheet information"`
}

// WithSheetsExport registers the sheets_export tool
func WithSheetsExport() Option {
	return func(reg *registry) {
		sdkmcp.AddTool(reg.server, &sdkmcp.Tool{
			Name:        "sheets_export",
			Description: "Export job selections to Google Sheets via the sheets_client integrations",
		}, sheetsExport)
	}
}

func sheetsExport(ctx context.Context, req *sdkmcp.CallToolRequest, params *SheetsExportParams) (*sdkmcp.CallToolResult, any, error) {
	_ = ctx
	_ = req

	msg := fmt.Sprintf("[sheets_export] Stub implementation: job_ids=%v spreadsheet_id=%q tab=%q", params.JobIDs, params.Sheet.SpreadsheetID, params.Sheet.Tab)
	return textResult(msg), nil, nil
}
