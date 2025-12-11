package main

import (
	"context"
	"fmt"
	"log"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	// Hardcoded test data - each test is independent
	testJobID1 = "550e8400-e29b-41d4-a716-446655440001"
	testJobID2 = "550e8400-e29b-41d4-a716-446655440002"
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
	testGraphTool(ctx, session)

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
		log.Printf("✗ graph_tool (cypher count) failed: %v", err)
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
		log.Printf("✗ graph_tool (cypher labels) failed: %v", err)
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
		log.Printf("✗ graph_tool (job_id) failed: %v", err)
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
		log.Printf("✗ graph_tool (default) failed: %v", err)
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
		log.Printf("✗ graph_tool (cypher with filter) failed: %v", err)
		return
	}
	printResult(result5)

	fmt.Println("\ngraph_tool all tests passed")
}

func printResult(res *mcp.CallToolResult) {
	for _, c := range res.Content {
		if txt, ok := c.(*mcp.TextContent); ok {
			fmt.Println(txt.Text)
		}
	}
}
