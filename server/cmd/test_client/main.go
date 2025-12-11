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

	// List available tools
	//testListTools(ctx, session)

	// Run independent tests
	//testJobSearch(ctx, session)
	//testPersistKeywords(ctx, session)
	//testJobAnalysis(ctx, session)
	//testGraphTool(ctx, session)
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

	// Using hardcoded job IDs - independent of job_search
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
		// Expected to fail if jobs don't exist in Neo4j - that's ok for this test
		log.Printf("persist_keywords: %v (expected if test jobs don't exist)", err)
		return
	}

	printResult(result)
	fmt.Println("persist_keywords passed")
}

func testJobAnalysis(ctx context.Context, session *mcp.ClientSession) {
	fmt.Println("TEST: job_analysis")

	// Test 1: With hardcoded job IDs
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

	// Test 2: With empty job IDs (edge case)
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

	// Test 1: Custom Cypher query - count all jobs
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

	// Test 2: Custom Cypher query - get node labels and counts
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

	// Test 3: Job ID inspection (without cypher)
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

	// Test 4: Default graph inspection (no parameters)
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

	// Test 5: Custom Cypher with filters
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

	// Test 1: Export with job IDs
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

	//// Test 2: Export with explicit rows
	//fmt.Println("\n  Test 2: Export explicit rows")
	//fmt.Printf("    Spreadsheet ID: %s\n", spreadsheetID)
	//fmt.Printf("    Number of rows: 2\n")
	//params2 := &mcp.CallToolParams{
	//	Name: "sheets_export",
	//	Arguments: map[string]any{
	//		"rows": []map[string]any{
	//			{
	//				"title":    "Software Engineer",
	//				"company":  "Test Company",
	//				"location": "Portland, OR",
	//				"url":      "https://example.com/job/1",
	//				"status":   "applied",
	//				"notes":    "Go, Kubernetes, AWS",
	//			},
	//			{
	//				"title":    "Senior Developer",
	//				"company":  "Another Company",
	//				"location": "Remote",
	//				"url":      "https://example.com/job/2",
	//				"status":   "interviewing",
	//				"notes":    "Python, Machine Learning",
	//			},
	//		},
	//		"sheet": map[string]any{
	//			"spreadsheet_id": spreadsheetID,
	//			"tab":            "Sheet1",
	//		},
	//		"upsert":    false,
	//		"clear_tab": false,
	//	},
	//}
	//
	//result2, err := session.CallTool(ctx, params2)
	//if err != nil {
	//	log.Printf("sheets_export (explicit rows) failed with error: %v", err)
	//	if result2 != nil {
	//		printResult(result2)
	//	}
	//	return
	//}
	//printResult(result2)
	//fmt.Println("sheets_export (explicit rows) completed")
	//
	//// Test 3: Export with filters
	//fmt.Println("\n  Test 3: Export with filters")
	//params3 := &mcp.CallToolParams{
	//	Name: "sheets_export",
	//	Arguments: map[string]any{
	//		"job_ids": []string{testJobID1, testJobID2},
	//		"filter": map[string]any{
	//			"source": "adzuna",
	//		},
	//		"sheet": map[string]any{
	//			"spreadsheet_id": spreadsheetID,
	//			"tab":            "FilteredJobs",
	//		},
	//		"upsert":    false,
	//		"clear_tab": false,
	//	},
	//}
	//
	//result3, err := session.CallTool(ctx, params3)
	//if err != nil {
	//	log.Printf("sheets_export (with filters) failed: %v", err)
	//	if result3 != nil {
	//		printResult(result3)
	//	}
	//	return
	//}
	//printResult(result3)
	//if result3.IsError {
	//	fmt.Println("sheets_export (with filters) returned error")
	//	return
	//}
	//fmt.Println("sheets_export (with filters) passed")

	fmt.Println("\nsheets_export tests completed")
}

func printResult(res *mcp.CallToolResult) {
	for _, c := range res.Content {
		if txt, ok := c.(*mcp.TextContent); ok {
			fmt.Println(txt.Text)
		}
	}
}
