package tools

import (
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// textResult returns a text-only ToolResult
func textResult(msg string) *sdkmcp.CallToolResult {
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{
			&sdkmcp.TextContent{Text: msg},
		},
	}
}
