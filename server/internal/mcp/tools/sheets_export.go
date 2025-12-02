package tools

import (
	"context"
	"encoding/json"
)

type SheetsExport struct{}

var (
	sheetsExportInputSchema = json.RawMessage(`{
		"type": "object",
		"properties": {
			"job_ids": {
				"type": "array",
				"items": { "type": "string" }
			},
			"filters": {
				"type": "object",
				"description": "Optional job_search style filters when IDs are not provided"
			},
			"sheet": {
				"type": "object",
				"properties": {
					"spreadsheet_id": { "type": "string" },
					"tab": { "type": "string" }
				},
				"required": ["spreadsheet_id"]
			}
		},
		"anyOf": [
			{ "required": ["job_ids", "sheet"] },
			{ "required": ["filters", "sheet"] }
		]
	}`)

	sheetsExportResultSchema = json.RawMessage(`{
		"type": "object",
		"properties": {
			"rows_written": { "type": "integer" },
			"spreadsheet_id": { "type": "string" },
			"tab": { "type": "string" },
			"sync_cursor": { "type": "string" }
		}
	}`)
)

func NewSheetsExport() *SheetsExport {
	return &SheetsExport{}
}

func (t *SheetsExport) Name() string {
	return "sheets_export"
}

func (t *SheetsExport) Description() string {
	return "Exports job selections to Google Sheets via the sheets_client integrations"
}

func (t *SheetsExport) InputSchema() json.RawMessage {
	return sheetsExportInputSchema
}

func (t *SheetsExport) ResultSchema() json.RawMessage {
	return sheetsExportResultSchema
}

func (t *SheetsExport) Execute(_ context.Context, params json.RawMessage) (any, error) {
	return stubResponse(
		t.Name(),
		"Stub implementation: fetch jobs, map via model_mapping, and upsert rows using sheets_client",
	), nil
}
