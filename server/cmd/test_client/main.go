package main

import (
	"context"
	"fmt"
	"log"
	"os"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	testJobID1 = "bf2bae05-b452-4413-a852-e0ee8f0562b8"
	testJobID2 = "4ac07b06-bac0-428d-ba63-1f88318ac81f"
)

func main() {
	ctx := context.Background()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "project-ets-test-client",
		Version: "0.1.0",
	}, nil)

	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: "http://localhost:8080/mcp/stream",
	}, nil)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer func() { _ = session.Close() }()

	log.Printf("Connected to server (session ID: %s)\n", session.ID())

	testSheetsExport(ctx, session)

	fmt.Println("\nAll tests completed")
}

func testJobSearch(ctx context.Context, session *mcp.ClientSession) {
	fmt.Println("\nTEST: job_search")

	params := &mcp.CallToolParams{
		Name: "job_search",
		Arguments: map[string]any{
			"query":    "software engineer",
			"location": "Portland",
		},
	}

	result, err := session.CallTool(ctx, params)
	if err != nil {
		log.Printf("job_search failed: %v", err)
		return
	}

	printResult(result)
	fmt.Println("job_search passed")
}

func testPersistKeywords(ctx context.Context, session *mcp.ClientSession) {
	fmt.Println("\nTEST: persist_keywords")

	params := &mcp.CallToolParams{
		Name: "persist_keywords",
		Arguments: map[string]any{
			"records": []map[string]any{
				{
					"job_id": testJobID1,
					"keywords": []map[string]any{
						{"value": "golang"},
						{"value": "kubernetes"},
						{"value": "microservices"},
					},
					"source": "test-client",
				},
				{
					"job_id": testJobID2,
					"keywords": []map[string]any{
						{"value": "python"},
						{"value": "machine-learning"},
					},
					"source": "test-client",
				},
			},
		},
	}

	result, err := session.CallTool(ctx, params)
	if err != nil {
		log.Printf("persist_keywords: %v (expected if test jobs don't exist)", err)
		return
	}

	printResult(result)
	fmt.Println("persist_keywords passed")
}

func testJobAnalysis(ctx context.Context, session *mcp.ClientSession) {
	fmt.Println("TEST: job_analysis")

	params := &mcp.CallToolParams{
		Name: "job_analysis",
		Arguments: map[string]any{
			"job_ids": []string{testJobID1, testJobID2},
			"profile": "Senior developer with 5 years Go, Kubernetes, AWS experience",
			"focus":   "skill gaps",
		},
	}

	result, err := session.CallTool(ctx, params)
	if err != nil {
		log.Printf("job_analysis failed: %v", err)
		return
	}

	printResult(result)

	fmt.Println("job_analysis with empty IDs")
	paramsEmpty := &mcp.CallToolParams{
		Name: "job_analysis",
		Arguments: map[string]any{
			"job_ids": []string{},
			"profile": "Test profile",
		},
	}

	resultEmpty, err := session.CallTool(ctx, paramsEmpty)
	if err != nil {
		log.Printf("job_analysis (empty) failed: %v", err)
		return
	}

	printResult(resultEmpty)
	fmt.Println("job_analysis passed")
}

func testGraphTool(ctx context.Context, session *mcp.ClientSession) {
	fmt.Println("\nTEST: graph_tool")

	fmt.Println("\n  Test 1: Custom Cypher query (count jobs)")
	params1 := &mcp.CallToolParams{
		Name: "graph_tool",
		Arguments: map[string]any{
			"cypher": "MATCH (j:Job) RETURN count(j) as total",
		},
	}

	result1, err := session.CallTool(ctx, params1)
	if err != nil {
		log.Printf("graph_tool (cypher count) failed: %v", err)
		return
	}
	printResult(result1)

	fmt.Println("\n  Test 2: Custom Cypher query (node labels)")
	params2 := &mcp.CallToolParams{
		Name: "graph_tool",
		Arguments: map[string]any{
			"cypher": "MATCH (n) RETURN labels(n) as labels, count(n) as count ORDER BY count DESC LIMIT 10",
		},
	}

	result2, err := session.CallTool(ctx, params2)
	if err != nil {
		log.Printf("graph_tool (cypher labels) failed: %v", err)
		return
	}
	printResult(result2)

	fmt.Println("\n  Test 3: Job ID inspection")
	params3 := &mcp.CallToolParams{
		Name: "graph_tool",
		Arguments: map[string]any{
			"job_id": testJobID1,
		},
	}

	result3, err := session.CallTool(ctx, params3)
	if err != nil {
		log.Printf("graph_tool (job_id) failed: %v", err)
		return
	}
	printResult(result3)

	fmt.Println("\n  Test 4: Default graph inspection")
	params4 := &mcp.CallToolParams{
		Name:      "graph_tool",
		Arguments: map[string]any{},
	}

	result4, err := session.CallTool(ctx, params4)
	if err != nil {
		log.Printf("graph_tool (default) failed: %v", err)
		return
	}
	printResult(result4)

	fmt.Println("\n  Test 5: Custom Cypher with job_id filter")
	params5 := &mcp.CallToolParams{
		Name: "graph_tool",
		Arguments: map[string]any{
			"cypher": "MATCH (j:Job {id: $jobId})-[:REQUIRES]->(s:Skill) RETURN j.title as title, collect(s.name) as skills",
			"job_id": testJobID1,
		},
	}

	result5, err := session.CallTool(ctx, params5)
	if err != nil {
		log.Printf("graph_tool (cypher with filter) failed: %v", err)
		return
	}
	printResult(result5)

	fmt.Println("\ngraph_tool all tests passed")
}

func testSheetsExport(ctx context.Context, session *mcp.ClientSession) {
	fmt.Println("\nTEST: sheets_export")

	spreadsheetID := os.Getenv("TEST_SPREADSHEET_ID")
	if spreadsheetID == "" {
		fmt.Println("TEST_SPREADSHEET_ID not set. Set it to your Google Sheet ID.")
		fmt.Println("Example: export TEST_SPREADSHEET_ID='your-spreadsheet-id-here'")
		return
	}

	fmt.Println("\n  Test 1: Export jobs by ID")
	params1 := &mcp.CallToolParams{
		Name: "sheets_export",
		Arguments: map[string]any{
			"job_ids": []string{testJobID1, testJobID2},
			"sheet": map[string]any{
				"spreadsheet_id": spreadsheetID,
				"tab":            "Sheet1",
			},
			"upsert":    false,
			"clear_tab": false,
		},
	}

	result1, err := session.CallTool(ctx, params1)
	if err != nil {
		log.Printf("sheets_export (job IDs) failed: %v", err)
	} else {
		printResult(result1)
		fmt.Println("sheets_export (job IDs) passed")
	}

	fmt.Println("\nsheets_export tests completed")
}

func printResult(res *mcp.CallToolResult) {
	for _, c := range res.Content {
		if txt, ok := c.(*mcp.TextContent); ok {
			fmt.Println(txt.Text)
		}
	}
}
