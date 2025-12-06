package main

import (
	"context"
	"fmt"
	"log"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// This is the test client for testing (no way)
// taken from official examples: https://github.com/modelcontextprotocol/go-sdk/blob/main/examples/http/main.go
func main() {
	ctx := context.Background()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "project-ets-test-client",
		Version: "0.1.0",
	}, nil)

	// Connect over streamable HTTP to localhost:8080
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: "http://localhost:8080/mcp/stream",
	}, nil)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			log.Printf("error closing MCP session: %v", err)
		}
	}()

	log.Printf("Connected to server (session ID: %s)", session.ID())

	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}
	fmt.Println("Tools:")
	for _, t := range toolsResult.Tools {
		fmt.Printf("  - %s: %s\n", t.Name, t.Description)
	}

	params := &mcp.CallToolParams{
		Name: "job_search",
		Arguments: map[string]any{
			"query":    "software engineer",
			"location": "Oregon",
		},
	}
	res, err := session.CallTool(ctx, params)
	if err != nil {
		log.Fatalf("CallTool job_search failed: %v", err)
	}

	fmt.Println("\njob_search result:")
	for _, c := range res.Content {
		if txt, ok := c.(*mcp.TextContent); ok {
			fmt.Println("  ", txt.Text)
		}
	}
}
