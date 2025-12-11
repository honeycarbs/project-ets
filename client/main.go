package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/generative-ai-go/genai"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/api/option"
)

const (
	systemPromptTemplate = `You are a professional job search assistant helping users manage their job search efficiently.

YOUR ROLE:
- Help users search for jobs, analyze opportunities, and export data to spreadsheets
- Understand natural language requests and select the appropriate tools
- Be proactive but ask clarifying questions when needed

AVAILABLE TOOLS:
- job_search: Find job postings based on search criteria
- persist_keywords: Extract and store ATS keywords from job descriptions
- job_analysis: Analyze job requirements and match with user profile using Graph RAG
- graph_tool: Query the job database for statistics and insights
- sheets_export: Export job data to Google Sheets%s

TOOL USAGE GUIDELINES:

For job searches ("find jobs", "search for", "show me jobs"):
1. Call job_search to retrieve job postings
2. AUTOMATICALLY extract 5-10 relevant ATS keywords from each job description
   - Focus on: technologies, programming languages, tools, frameworks, methodologies, certifications
   - Think like an ATS system: what would a recruiter search for?
3. Call persist_keywords with ALL extracted keywords at once
4. If sheets are configured, call sheets_export to save the results
5. Provide a summary to the user

For job analysis ("analyze job", "tell me about job", "evaluate position"):
- ONLY call job_analysis with the job ID
- Do NOT extract keywords or search for additional jobs
- Focus on providing insights about that specific job

For statistics ("how many jobs", "show me trends", "what's the count"):
- ONLY call graph_tool to query the database
- Do NOT search for jobs or extract keywords
- Provide clear statistical answers

For general questions about your role or capabilities:
- Answer directly without calling tools
- Be helpful and explain what you can do

IMPORTANT RULES:
1. Choose the right tool for the task - don't over-complicate simple requests
2. For job_search, keyword extraction is MANDATORY (automatic step 2)
3. For job_analysis or graph_tool, do NOT extract keywords
4. If a tool call fails, explain the error clearly and offer alternatives
5. Don't ask permission to extract keywords during job search - just do it
6. Never make up data - only use information from tool responses

ERROR HANDLING:
- If a tool fails, explain what went wrong in plain language
- Suggest what the user should do next
- Don't give up after one failure - try alternative approaches if reasonable

Remember: You are here to make the job search process smooth and efficient. Be proactive, accurate, and helpful.`
)

type Client struct {
	mcpSession             *mcp.ClientSession
	gemini                 *genai.Client
	model                  *genai.GenerativeModel
	tools                  []*mcp.Tool
	lastToolCalled         string
	jobSearchResultPending bool
	sheetsID               string
}

func NewClient(ctx context.Context, mcpEndpoint, apiKey, model, sheetsID string) (*Client, error) {
	// Connect to MCP server
	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "project-ets-client",
		Version: "0.1.0",
	}, nil)

	fmt.Printf("Connecting to MCP server at: %s\n", mcpEndpoint)
	
	session, err := mcpClient.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint: mcpEndpoint,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server at %s: %w", mcpEndpoint, err)
	}
	
	fmt.Printf("Successfully connected (session ID: %s)\n", session.ID())

	// Initialize Gemini AI
	geminiClient, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to initialize Gemini: %w", err)
	}

	geminiModel := geminiClient.GenerativeModel(model)
	
	// Build system prompt with sheets ID if available
	var systemPrompt string
	if sheetsID != "" {
		sheetsInstruction := fmt.Sprintf("\n\nFor sheets_export, ALWAYS use this Google Sheets ID: %s\nFormat: {\"job_ids\": [\"id1\", \"id2\"], \"sheet\": {\"spreadsheet_id\": \"%s\", \"tab\": \"Sheet1\"}}\nDO NOT ask the user for the spreadsheet ID.", sheetsID, sheetsID)
		systemPrompt = fmt.Sprintf(systemPromptTemplate, sheetsInstruction)
	} else {
		systemPrompt = fmt.Sprintf(systemPromptTemplate, "")
	}
	
	// Set system instruction directly on the model
	geminiModel.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
	}

	client := &Client{
		mcpSession: session,
		gemini:     geminiClient,
		model:      geminiModel,
		sheetsID:   sheetsID,
	}

	// List available tools
	toolsResp, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	client.tools = toolsResp.Tools

	return client, nil
}

func (c *Client) Close() error {
	var errs []error
	
	if err := c.gemini.Close(); err != nil {
		errs = append(errs, fmt.Errorf("gemini close error: %w", err))
	}
	
	if err := c.mcpSession.Close(); err != nil {
		errs = append(errs, fmt.Errorf("mcp session close error: %w", err))
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("errors during close: %v", errs)
	}
	
	return nil
}

func (c *Client) RunQuery(ctx context.Context, userQuery string) error {
	fmt.Printf("\nUser Query: %s\n\n", userQuery)
	fmt.Println("Agent is processing your request...\n")
	fmt.Println(strings.Repeat("=", 80) + "\n")

	// Reset workflow tracking for new query
	c.lastToolCalled = ""
	c.jobSearchResultPending = false

	// Configure tools on the model
	tools := c.buildGeminiTools()
	if len(tools) > 0 {
		c.model.Tools = tools
	}

	// Build conversation with Gemini
	chat := c.model.StartChat()

	// Send initial message
	maxIterations := 10
	iteration := 0
	var currentParts []genai.Part
	currentParts = append(currentParts, genai.Text(userQuery))

	for iteration < maxIterations {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		iteration++

		if iteration == 1 {
			fmt.Println("[Agent] Analyzing your request...")
		} else {
			fmt.Printf("[Agent] Processing step %d...\n", iteration)
		}

		// Send message to Gemini
		resp, err := chat.SendMessage(ctx, currentParts...)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("gemini API error: %w", err)
		}

		// Process response
		hasFunctionCall := false
		var responseText strings.Builder
		var functionResponses []genai.Part
		
		for _, candidate := range resp.Candidates {
			if candidate.Content == nil {
				continue
			}
			
			for _, part := range candidate.Content.Parts {
				// Check for function call
				fc, isFunctionCall := part.(genai.FunctionCall)
				if isFunctionCall {
					hasFunctionCall = true
					toolName := fc.Name
					args := fc.Args
					if args == nil {
						args = make(map[string]interface{})
					}

					// WORKFLOW ENFORCEMENT: Block sheets_export if keywords weren't extracted
					if toolName == "sheets_export" && c.jobSearchResultPending {
						fmt.Println("\n[Workflow Error] Cannot export to sheets without extracting keywords first!")
						
						errorResult := map[string]interface{}{
							"error": "Workflow violation: You must call persist_keywords to extract and store keywords from the job search results BEFORE calling sheets_export.",
							"required_action": "Call persist_keywords with all job IDs and their extracted keywords",
						}
						
						functionResponses = append(functionResponses, genai.FunctionResponse{
							Name:     toolName,
							Response: errorResult,
						})
						continue
					}

					// Show progress
					showToolProgress(toolName)

					// Call the MCP tool
					result, err := c.callMCPTool(ctx, toolName, args)
					if err != nil {
						if ctx.Err() != nil {
							return ctx.Err()
						}
						fmt.Printf("[Error] Tool error: %v\n", err)
						
						errorResult := map[string]interface{}{
							"error": err.Error(),
						}
						functionResponses = append(functionResponses, genai.FunctionResponse{
							Name:     toolName,
							Response: errorResult,
						})
						continue
					}

					// Track workflow state
					c.lastToolCalled = toolName
					if toolName == "job_search" {
						c.jobSearchResultPending = true
					} else if toolName == "persist_keywords" {
						c.jobSearchResultPending = false
					}

					fmt.Printf("[Success] %s completed\n", getToolDisplayName(toolName))

					// Add function response
					functionResponses = append(functionResponses, genai.FunctionResponse{
						Name:     toolName,
						Response: result,
					})
				} else if textPart, ok := part.(genai.Text); ok {
					responseText.WriteString(string(textPart))
				}
			}
		}

		// If we have function calls, send all responses back at once
		if hasFunctionCall && len(functionResponses) > 0 {
			// Send all function responses together as Parts
			currentParts = functionResponses
			continue
		}

		// Check for final text response
		if responseText.Len() > 0 {
			fmt.Println("\n" + strings.Repeat("=", 80))
			fmt.Println("FINAL RESPONSE:")
			fmt.Println(strings.Repeat("=", 80) + "\n")
			fmt.Printf("%s\n\n", responseText.String())
			fmt.Println(strings.Repeat("=", 80) + "\n")
			return nil
		}

		// No function calls and no text - might be waiting
		if len(resp.Candidates) > 0 {
			continue
		}
		
		return fmt.Errorf("unexpected response format from Gemini")
	}

	return fmt.Errorf("max iterations reached")
}

func showToolProgress(toolName string) {
	switch toolName {
	case "job_search":
		fmt.Println("\n[Tool] Searching for jobs...")
	case "persist_keywords":
		fmt.Println("\n[Tool] Extracting and storing keywords from job descriptions...")
	case "sheets_export":
		fmt.Println("\n[Tool] Exporting jobs to Google Sheets...")
	case "job_analysis":
		fmt.Println("\n[Tool] Analyzing jobs using Graph RAG...")
	case "graph_tool":
		fmt.Println("\n[Tool] Querying job database...")
	default:
		fmt.Printf("\n[Tool] Calling tool: %s...\n", toolName)
	}
}

func (c *Client) buildGeminiTools() []*genai.Tool {
	var geminiTools []*genai.Tool

	for _, tool := range c.tools {
		declaration := &genai.FunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
		}

		if tool.InputSchema != nil {
			schema := convertSchema(tool.InputSchema)
			declaration.Parameters = schema
		} else {
			declaration.Parameters = &genai.Schema{
				Type: genai.TypeObject,
			}
		}

		geminiTool := &genai.Tool{
			FunctionDeclarations: []*genai.FunctionDeclaration{declaration},
		}
		geminiTools = append(geminiTools, geminiTool)
	}

	return geminiTools
}

func convertSchema(schema interface{}) *genai.Schema {
	if schema == nil {
		return &genai.Schema{
			Type: genai.TypeObject,
		}
	}

	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		return &genai.Schema{
			Type: genai.TypeObject,
		}
	}

	result := &genai.Schema{
		Properties: make(map[string]*genai.Schema),
	}

	if typeVal, ok := schemaMap["type"].(string); ok {
		switch typeVal {
		case "string":
			result.Type = genai.TypeString
		case "number", "integer":
			result.Type = genai.TypeNumber
		case "boolean":
			result.Type = genai.TypeBoolean
		case "array":
			result.Type = genai.TypeArray
		case "object":
			result.Type = genai.TypeObject
		default:
			result.Type = genai.TypeObject
		}
	} else {
		result.Type = genai.TypeObject
	}

	if required, ok := schemaMap["required"].([]interface{}); ok {
		for _, req := range required {
			if reqStr, ok := req.(string); ok {
				result.Required = append(result.Required, reqStr)
			}
		}
	}

	if properties, ok := schemaMap["properties"].(map[string]interface{}); ok {
		for name, prop := range properties {
			result.Properties[name] = convertSchema(prop)
		}
	}

	if items, ok := schemaMap["items"]; ok {
		result.Items = convertSchema(items)
	}

	return result
}

func (c *Client) callMCPTool(ctx context.Context, toolName string, args map[string]interface{}) (map[string]interface{}, error) {
	var tool *mcp.Tool
	for _, t := range c.tools {
		if t.Name == toolName {
			tool = t
			break
		}
	}

	if tool == nil {
		return nil, fmt.Errorf("tool %s not found", toolName)
	}

	toolCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			cancel()
		case <-done:
		}
	}()
	defer close(done)

	result, err := c.mcpSession.CallTool(toolCtx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		return nil, err
	}

	resultMap := make(map[string]interface{})
	var resultTexts []string

	for _, content := range result.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			resultTexts = append(resultTexts, textContent.Text)
		}
	}

	if len(resultTexts) > 0 {
		resultMap["result"] = strings.Join(resultTexts, "\n")
	} else {
		resultMap["result"] = "Tool executed successfully"
	}

	return resultMap, nil
}

func getToolDisplayName(toolName string) string {
	switch toolName {
	case "job_search":
		return "Job search"
	case "persist_keywords":
		return "Keyword extraction"
	case "sheets_export":
		return "Google Sheets export"
	case "job_analysis":
		return "Job analysis"
	case "graph_tool":
		return "Database query"
	default:
		return toolName
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\nShutting down...")
		cancel()
		
		go func() {
			time.Sleep(1 * time.Second)
			os.Exit(0)
		}()
	}()

	mcpEndpoint := os.Getenv("MCP_URL")
	if mcpEndpoint == "" {
		mcpEndpoint = "http://localhost:8080"
	}
	
	if !strings.HasSuffix(mcpEndpoint, "/mcp/stream") {
		mcpEndpoint = strings.TrimSuffix(mcpEndpoint, "/")
		mcpEndpoint = mcpEndpoint + "/mcp/stream"
	}

	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		log.Fatal("GOOGLE_API_KEY or GEMINI_API_KEY environment variable must be set")
	}

	model := os.Getenv("GOOGLE_MODEL")
	if model == "" {
		model = "gemini-2.5-flash"
	}
	
	sheetsID := os.Getenv("GOOGLE_SHEETS_ID")
	if sheetsID == "" {
		sheetsID = os.Getenv("SHEETS_ID")
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("ETS MCP CLIENT CONFIGURATION")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("MCP Server URL: %s\n", mcpEndpoint)
	fmt.Printf("Google Model: %s\n", model)
	fmt.Printf("API Key: Set\n")
	if sheetsID != "" {
		fmt.Printf("Google Sheets ID: %s\n", sheetsID)
	} else {
		fmt.Println("Google Sheets ID: Not set (will prompt when needed)")
	}
	fmt.Println(strings.Repeat("=", 80))

	client, err := NewClient(ctx, mcpEndpoint, apiKey, model, sheetsID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer func() {
		closeDone := make(chan struct{})
		go func() {
			client.Close()
			close(closeDone)
		}()
		
		select {
		case <-closeDone:
		case <-time.After(500 * time.Millisecond):
			fmt.Println("Warning: Client close timed out")
		}
	}()

	fmt.Printf("\nConnected to MCP server (session ID: %s)\n", client.mcpSession.ID())
	fmt.Printf("Loaded %d tools from ETS server\n\n", len(client.tools))

	fmt.Println("Available Tools:")
	for i, tool := range client.tools {
		fmt.Printf("  %d. %s", i+1, tool.Name)
		if tool.Description != "" {
			desc := strings.ReplaceAll(tool.Description, "\n", " ")
			if len(desc) > 100 {
				desc = desc[:100] + "..."
			}
			fmt.Printf(" - %s", desc)
		}
		fmt.Println()
	}
	fmt.Println()

	if len(os.Args) > 1 {
		query := strings.Join(os.Args[1:], " ")
		if err := client.RunQuery(ctx, query); err != nil {
			log.Fatalf("Error: %v", err)
		}
	} else {
		fmt.Println(strings.Repeat("=", 80))
		fmt.Println("ETS MCP CLIENT - INTERACTIVE MODE")
		fmt.Println(strings.Repeat("=", 80))
		fmt.Println("\nWelcome to the Employer Tracking System!")
		fmt.Println("This client connects to your ETS MCP server to help you:")
		fmt.Println("  - Search for job opportunities")
		fmt.Println("  - Extract and store relevant keywords")
		fmt.Println("  - Analyze job matches")
		fmt.Println("  - Export jobs to Google Sheets")
		fmt.Println("\nType 'quit' or 'exit' to end the session.")
		fmt.Println(strings.Repeat("=", 80) + "\n")

		scanner := bufio.NewScanner(os.Stdin)
		inputChan := make(chan string)
		go func() {
			for scanner.Scan() {
				inputChan <- scanner.Text()
			}
			close(inputChan)
		}()
		
		for {
			fmt.Print("\nYour request: ")
			
			select {
			case <-ctx.Done():
				fmt.Println("\n\nShutdown complete.")
				return
				
			case input, ok := <-inputChan:
				if !ok {
					return
				}
				
				input = strings.TrimSpace(input)
				if input == "" {
					continue
				}

				if strings.ToLower(input) == "quit" || strings.ToLower(input) == "exit" || strings.ToLower(input) == "q" {
					fmt.Println("\nThank you for using ETS! Goodbye.\n")
					return
				}

				if err := client.RunQuery(ctx, input); err != nil {
					if err == context.Canceled || err == context.DeadlineExceeded {
						return
					}
					fmt.Printf("\nAn error occurred: %v\n", err)
				}
			}
		}
	}
}