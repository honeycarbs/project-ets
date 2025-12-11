package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/honeycarbs/project-ets/internal/domain"
	"github.com/honeycarbs/project-ets/internal/repository"
	"github.com/honeycarbs/project-ets/pkg/logging"
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

// SheetsClient exports rows to Google Sheets
type SheetsClient interface {
	Export(ctx context.Context, params SheetsExportParams) (SheetsExportResult, error)
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

type sheetsExportTool struct {
	client SheetsClient
	repo   repository.JobRepository
	logger *logging.Logger
}

// WithSheetsExport registers the sheets_export tool
func WithSheetsExport(client SheetsClient, logger *logging.Logger) Option {
	return func(reg *registry) {
		handler := sheetsExportTool{client: client, logger: logger}
		sdkmcp.AddTool(reg.server, &sdkmcp.Tool{
			Name:        "sheets_export",
			Description: "Export job selections to Google Sheets via the sheets_client integrations",
		}, handler.handle)
	}
}

func RegisterExportTools(server *sdkmcp.Server, client SheetsClient, repo repository.JobRepository, logger *logging.Logger) error {
	handler := sheetsExportTool{client: client, repo: repo, logger: logger}
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "sheets_export",
		Description: "Export job selections to Google Sheets via the sheets_client integrations",
	}, handler.handle)
	return nil
}

func (t sheetsExportTool) handle(ctx context.Context, req *sdkmcp.CallToolRequest, params *SheetsExportParams) (*sdkmcp.CallToolResult, any, error) {
	_ = req

	if params == nil {
		t.logWarn("sheets_export: parameters are required")
		return textResult("sheets_export: parameters are required"), SheetsExportResult{}, fmt.Errorf("parameters are required")
	}

	if params.Sheet.SpreadsheetID == "" {
		t.logWarn("sheets_export: spreadsheet_id is required")
		return textResult("sheets_export: spreadsheet_id is required"), SheetsExportResult{}, fmt.Errorf("spreadsheet_id is required")
	}

	t.logInfo("sheets_export start",
		"spreadsheet_id", params.Sheet.SpreadsheetID,
		"tab", params.Sheet.Tab,
		"job_ids", len(params.JobIDs),
		"rows", len(params.Rows),
		"upsert", params.Upsert,
		"clear_tab", params.ClearTab,
	)

	var rows []SheetRow
	var mode string

	if len(params.JobIDs) > 0 {
		t.logInfo("sheets_export hydrate_jobs", "job_ids", params.JobIDs)
		jobRows, err := t.fetchJobsAsRows(ctx, params.JobIDs, params.Filter)
		if err != nil {
			t.logError("sheets_export fetchJobsAsRows failed", "err", err)
			return textResult(fmt.Sprintf("sheets_export: failed to fetch jobs: %v", err)), SheetsExportResult{}, err
		}
		rows = jobRows
		mode = "hydrate_jobs"
	} else if len(params.Rows) > 0 {
		t.logInfo("sheets_export append_rows", "provided_rows", len(params.Rows))
		rows = params.Rows
		mode = "append_rows"
	} else {
		t.logWarn("sheets_export: either job_ids or rows must be provided")
		return textResult("sheets_export: either job_ids or rows must be provided"), SheetsExportResult{}, fmt.Errorf("either job_ids or rows must be provided")
	}

	if len(rows) == 0 && mode == "append_rows" {
		t.logWarn("sheets_export: received empty rows in append mode", "provided_rows", len(params.Rows))
		return textResult(fmt.Sprintf("sheets_export: received %d rows in params (expected > 0)", len(params.Rows))), SheetsExportResult{}, fmt.Errorf("no rows to export")
	}

	if len(rows) == 0 {
		t.logWarn("sheets_export: no rows after hydration", "mode", mode)
		result := SheetsExportResult{
			SpreadsheetID: params.Sheet.SpreadsheetID,
			Tab:           params.Sheet.Tab,
			WrittenRows:   0,
			Mode:          "noop",
			CompletedAt:   time.Now().UTC(),
			Message:       "no rows to export",
		}
		return textResult("[sheets_export] No rows to export"), result, nil
	}

	exportParams := SheetsExportParams{
		JobIDs:   params.JobIDs,
		Rows:     rows,
		Filter:   params.Filter,
		Upsert:   params.Upsert,
		ClearTab: params.ClearTab,
		Sheet:    params.Sheet,
		Metadata: params.Metadata,
	}

	result, err := t.client.Export(ctx, exportParams)
	if err != nil {
		t.logError("sheets_export: export failed", "err", err)
		return textResult(fmt.Sprintf("sheets_export: export failed: %v", err)), SheetsExportResult{}, err
	}

	result.Mode = mode
	if result.CompletedAt.IsZero() {
		result.CompletedAt = time.Now().UTC()
	}
	
	// Preserve spreadsheet ID and tab from params if result doesn't have them
	if result.SpreadsheetID == "" {
		result.SpreadsheetID = params.Sheet.SpreadsheetID
	}
	if result.Tab == "" {
		result.Tab = params.Sheet.Tab
	}

	t.logInfo("sheets_export complete",
		"mode", result.Mode,
		"written_rows", result.WrittenRows,
		"spreadsheet_id", result.SpreadsheetID,
		"tab", result.Tab,
	)

	msg := fmt.Sprintf("[sheets_export] Exported %d row(s) to spreadsheet %q (tab: %q, mode: %s)", result.WrittenRows, result.SpreadsheetID, result.Tab, result.Mode)
	return textResult(msg), result, nil
}

func (t sheetsExportTool) fetchJobsAsRows(ctx context.Context, jobIDs []string, filter map[string]string) ([]SheetRow, error) {
	if t.repo == nil {
		t.logWarn("sheets_export: job repository not available")
		return nil, fmt.Errorf("job repository not available")
	}

	ids := make([]domain.JobID, 0, len(jobIDs))
	for _, idStr := range jobIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			t.logWarn("sheets_export: skipping invalid job id", "job_id", idStr)
			continue
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		t.logWarn("sheets_export: no valid job IDs provided", "job_ids", jobIDs)
		return nil, fmt.Errorf("no valid job IDs provided")
	}

	t.logInfo("sheets_export: fetching jobs", "count", len(ids))
	jobs, err := t.repo.FindByIDs(ctx, ids)
	if err != nil {
		t.logError("sheets_export: failed to fetch jobs", "err", err)
		return nil, fmt.Errorf("failed to fetch jobs: %w", err)
	}

	t.logInfo("sheets_export: jobs fetched", "jobs", len(jobs))
	rows := make([]SheetRow, 0, len(jobs))
	for _, job := range jobs {
		if !t.matchesFilter(job, filter) {
			continue
		}

		row := SheetRow{
			Title:    job.Title,
			Company:  job.Company.Name,
			Location: job.Location,
			URL:      job.URL,
		}

		if len(job.Skills) > 0 {
			skillNames := make([]string, 0, len(job.Skills))
			for _, skill := range job.Skills {
				skillNames = append(skillNames, skill.Name)
			}
			row.Notes = strings.Join(skillNames, ", ")
		}

		rows = append(rows, row)
	}

	return rows, nil
}

func (t sheetsExportTool) matchesFilter(job domain.Job, filter map[string]string) bool {
	if len(filter) == 0 {
		return true
	}

	for key, value := range filter {
		switch key {
		case "source":
			if job.Source != value {
				return false
			}
		case "location":
			if !strings.Contains(strings.ToLower(job.Location), strings.ToLower(value)) {
				return false
			}
		case "company":
			if !strings.Contains(strings.ToLower(job.Company.Name), strings.ToLower(value)) {
				return false
			}
		case "remote":
			if value == "true" && !job.Remote {
				return false
			}
			if value == "false" && job.Remote {
				return false
			}
		}
	}

	return true
}

func (t sheetsExportTool) logInfo(msg string, args ...any) {
	if t.logger != nil {
		t.logger.Info(msg, args...)
	}
}

func (t sheetsExportTool) logWarn(msg string, args ...any) {
	if t.logger != nil {
		t.logger.Warn(msg, args...)
	}
}

func (t sheetsExportTool) logError(msg string, args ...any) {
	if t.logger != nil {
		t.logger.Error(msg, args...)
	}
}
