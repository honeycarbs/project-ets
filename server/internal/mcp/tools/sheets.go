package tools

import (
	"context"
	"fmt"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// SheetRow defines a row to upsert into Sheets
type SheetRow struct {
	Title     string `json:"title,omitempty" jsonschema:"Job title text"`
	Company   string `json:"company,omitempty" jsonschema:"Company name"`
	Location  string `json:"location,omitempty" jsonschema:"Location text"`
	URL       string `json:"url,omitempty" jsonschema:"Application URL"`
	Status    string `json:"status,omitempty" jsonschema:"Pipeline status e.g. applied/interviewing"`
	Color     string `json:"color,omitempty" jsonschema:"Row highlight color (hex or named)"`
	Notes     string `json:"notes,omitempty" jsonschema:"Free-form notes or instructions"`
	UpdatedAt string `json:"updated_at,omitempty" jsonschema:"ISO timestamp captured by client"`
}

// SheetsExportParams defines the arguments for the sheets_export tool
type SheetsExportParams struct {
	JobIDs   []string          `json:"job_ids,omitempty" jsonschema:"Jobs to rehydrate from storage"`
	Rows     []SheetRow        `json:"rows,omitempty" jsonschema:"Explicit rows to write when not rehydrating"`
	Filter   map[string]string `json:"filter,omitempty" jsonschema:"Optional filter tags applied server-side"`
	Upsert   bool              `json:"upsert,omitempty" jsonschema:"Whether to upsert (true) or append (false)"`
	ClearTab bool              `json:"clear_tab,omitempty" jsonschema:"If true, clears the tab before writing"`
	Sheet    struct {
		SpreadsheetID string `json:"spreadsheet_id" jsonschema:"Google Sheets document ID"`
		Tab           string `json:"tab,omitempty" jsonschema:"Tab name to upsert data"`
		Range         string `json:"range,omitempty" jsonschema:"Optional A1 range override"`
	} `json:"sheet" jsonschema:"Destination sheet information"`
	Metadata map[string]string `json:"metadata,omitempty" jsonschema:"Optional agent metadata (e.g., run_id)"`
}

// SheetsExportResult describes the summary returned after export
type SheetsExportResult struct {
	SpreadsheetID string    `json:"spreadsheet_id" jsonschema:"Target spreadsheet ID"`
	Tab           string    `json:"tab,omitempty" jsonschema:"Target tab name"`
	WrittenRows   int       `json:"written_rows" jsonschema:"How many rows were written"`
	Mode          string    `json:"mode" jsonschema:"append, upsert, or hydrate"`
	CompletedAt   time.Time `json:"completed_at" jsonschema:"Timestamp when export finished"`
	Message       string    `json:"message,omitempty" jsonschema:"Optional status message"`
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

	result := SheetsExportResult{}
	if params != nil {
		result.SpreadsheetID = params.Sheet.SpreadsheetID
		result.Tab = params.Sheet.Tab
		result.Mode = "unknown"

		switch {
		case len(params.Rows) > 0:
			result.WrittenRows = len(params.Rows)
			result.Mode = "append_rows"
		case len(params.JobIDs) > 0:
			result.WrittenRows = len(params.JobIDs)
			result.Mode = "hydrate_jobs"
		default:
			result.Mode = "noop"
		}
	}

	result.CompletedAt = time.Now().UTC()
	if result.Message == "" {
		result.Message = "sheets export stub executed"
	}

	msg := fmt.Sprintf("[sheets_export] Stub implementation: mode=%s spreadsheet_id=%q tab=%q", result.Mode, result.SpreadsheetID, result.Tab)
	return textResult(msg), result, nil
}
