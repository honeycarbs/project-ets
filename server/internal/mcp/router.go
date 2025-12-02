package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// Tool represents an MCP tool implementation.
type Tool interface {
	Name() string
	Description() string
	InputSchema() json.RawMessage
	ResultSchema() json.RawMessage
	Execute(ctx context.Context, params json.RawMessage) (any, error)
}

// Router stores and dispatches MCP tools.
type Router struct {
	tools map[string]Tool
}

// NewRouter creates an empty router instance.
func NewRouter() *Router {
	return &Router{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the router registry.
func (r *Router) Register(tool Tool) error {
	name := tool.Name()
	if _, ok := r.tools[name]; ok {
		return fmt.Errorf("tool %q already registered", name)
	}

	r.tools[name] = tool
	return nil
}

// List exposes MCP tool descriptors.
func (r *Router) List() []ToolInfo {
	out := make([]ToolInfo, 0, len(r.tools))
	for _, tool := range r.tools {
		out = append(out, ToolInfo{
			Name:         tool.Name(),
			Description:  tool.Description(),
			InputSchema:  tool.InputSchema(),
			ResultSchema: tool.ResultSchema(),
		})
	}

	return out
}

// Call executes a tool by name.
func (r *Router) Call(ctx context.Context, name string, params json.RawMessage) (any, error) {
	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool %q not found", name)
	}

	return tool.Execute(ctx, params)
}
