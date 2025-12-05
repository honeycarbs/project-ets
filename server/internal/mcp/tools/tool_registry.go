package tools

import (
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Option configures which tools are registered
type Option func(*registry)

type registry struct {
	server *sdkmcp.Server
}

// Register applies the provided tool options
func Register(server *sdkmcp.Server, opts ...Option) {
	reg := &registry{server: server}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(reg)
	}
}
