package tools

import "github.com/honeycarbs/project-ets/internal/mcp"

// Registrar describes the subset of MCP server needed to register tools.
type Registrar interface {
	RegisterTool(mcp.Tool) error
}

// RegisterAll installs every baseline tool into the provided registrar.
func RegisterAll(r Registrar) error {
	defaultTools := []mcp.Tool{
		NewJobSearch(),
		NewJobAnalysis(),
		NewGraphTool(),
		NewSheetsExport(),
	}

	for _, tool := range defaultTools {
		if err := r.RegisterTool(tool); err != nil {
			return err
		}
	}

	return nil
}
