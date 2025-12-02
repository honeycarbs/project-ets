package mcp

import (
	"encoding/json"
)

// RPCRequest represents a JSON-RPC 2.0 request
type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`          // must be "2.0"
	ID      json.RawMessage `json:"id,omitempty"`     // identifier echoed back in responses
	Method  string          `json:"method"`           // RPC method name such as initialize
	Params  json.RawMessage `json:"params,omitempty"` // raw payload passed to the method
}

// RPCResponse represents a JSON-RPC 2.0 response
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`     // mirrors request ID
	Result  any             `json:"result,omitempty"` // successful payload
	Error   *RPCError       `json:"error,omitempty"`  // populated when the call fails
}

// RPCError conveys JSON-RPC error information
type RPCError struct {
	Code    int         `json:"code"` // per JSON-RPC spec
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"` // optional extra info
}

// InitializeParams describes the MCP initialize call payload
type InitializeParams struct {
	ClientName    string `json:"client_name"`
	ClientVersion string `json:"client_version"` // semantic version reported by the client
}

// InitializeResult describes server capabilities returned to the MCP client
type InitializeResult struct {
	ServerInfo   ServerInfo   `json:"server_info"`
	Capabilities Capabilities `json:"capabilities"`
	Tools        []ToolInfo   `json:"tools,omitempty"` // optional eager tool advertisement
}

// ServerInfo provides metadata about this MCP server
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"` // semantic version of the server binary
}

type Capabilities struct {
	// TODO resource streaming, events, etc
}

// ListToolsResult enumerates every tool visible to the client
type ListToolsResult struct {
	Tools []ToolInfo `json:"tools"`
}

// ToolInfo describes an MCP tool and its schemas
type ToolInfo struct {
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`   // human friendly summary for clients
	InputSchema  json.RawMessage `json:"input_schema,omitempty"`  // JSON schema describing params
	ResultSchema json.RawMessage `json:"result_schema,omitempty"` // JSON schema describing responses
}

// CallToolParams is the payload for tool execution
type CallToolParams struct {
	Name   string          `json:"name"`
	Params json.RawMessage `json:"params,omitempty"` // raw tool parameters
}
