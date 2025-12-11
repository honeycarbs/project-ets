package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/honeycarbs/project-ets/internal/mcp/tools"
	sheetsclient "github.com/honeycarbs/project-ets/pkg/sheets"
)

type sheetsClientAdapter struct {
	client *sheetsclient.Client
}

func (a *sheetsClientAdapter) Export(ctx context.Context, params tools.SheetsExportParams) (tools.SheetsExportResult, error) {
	if a.client == nil {
		return tools.SheetsExportResult{
			SpreadsheetID: params.Sheet.SpreadsheetID,
			Tab:           params.Sheet.Tab,
			Message:       "Google Sheets client not configured (GOOGLE_SHEETS_CREDENTIALS_PATH not set)",
		}, fmt.Errorf("sheets: client not configured")
	}

	result := tools.SheetsExportResult{
		SpreadsheetID: params.Sheet.SpreadsheetID,
		Tab:           params.Sheet.Tab,
		CompletedAt:   time.Now().UTC(),
	}

	if len(params.Rows) == 0 {
		result.Message = "no rows to export"
		return result, nil
	}

	range_ := buildRange(params)
	values := convertRowsToValues(params.Rows)

	if params.ClearTab {
		clearRange := buildClearRange(params.Sheet.Tab)
		if err := a.client.ClearValues(ctx, params.Sheet.SpreadsheetID, clearRange); err != nil {
			return result, fmt.Errorf("sheets: failed to clear sheet: %w", err)
		}
	}

	if params.Upsert {
		if err := a.client.UpdateValues(ctx, params.Sheet.SpreadsheetID, range_, values); err != nil {
			return result, fmt.Errorf("sheets: failed to upsert rows: %w", err)
		}
	} else {
		if err := a.client.AppendValues(ctx, params.Sheet.SpreadsheetID, range_, values); err != nil {
			return result, fmt.Errorf("sheets: failed to append rows: %w", err)
		}
	}

	result.WrittenRows = len(params.Rows)
	result.Message = fmt.Sprintf("successfully exported %d row(s)", result.WrittenRows)

	return result, nil
}

func buildRange(params tools.SheetsExportParams) string {
	if params.Sheet.Range != "" {
		return params.Sheet.Range
	}

	tab := params.Sheet.Tab
	if tab == "" {
		tab = "Sheet1"
	}

	if params.Upsert {
		return fmt.Sprintf("%s!A2", tab)
	}
	return fmt.Sprintf("%s!A1", tab)
}

func buildClearRange(tab string) string {
	if tab == "" {
		tab = "Sheet1"
	}
	return fmt.Sprintf("%s!A2:Z", tab)
}

func convertRowsToValues(rows []tools.SheetRow) [][]interface{} {
	values := make([][]interface{}, len(rows))
	for i, row := range rows {
		values[i] = []interface{}{
			row.Title,
			row.Company,
			row.Location,
			row.URL,
			row.Status,
			row.Color,
			row.Notes,
			row.UpdatedAt,
		}
	}
	return values
}

