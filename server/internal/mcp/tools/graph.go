package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	pkgneo4j "github.com/honeycarbs/project-ets/pkg/neo4j"
)

// GraphToolParams defines the arguments for the graph_tool tool
type GraphToolParams struct {
	Cypher  string                 `json:"cypher,omitempty" jsonschema:"Custom Cypher query to run"`
	JobID   string                 `json:"job_id,omitempty"`
	UserID  string                 `json:"user_id,omitempty"`
	Filters map[string]interface{} `json:"filters,omitempty" jsonschema:"Optional label/relation filters"`
}

type graphToolHandler struct {
	client *pkgneo4j.Client
}

// WithGraphTool registers the graph_tool
func WithGraphTool(client *pkgneo4j.Client) Option {
	return func(reg *registry) {
		handler := graphToolHandler{client: client}
		sdkmcp.AddTool(reg.server, &sdkmcp.Tool{
			Name:        "graph_tool",
			Description: "Developer tool for inspecting and debugging the Neo4j knowledge graph",
		}, handler.handle)
	}
}

func RegisterGraphTool(server *sdkmcp.Server, client *pkgneo4j.Client) error {
	handler := graphToolHandler{client: client}
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "graph_tool",
		Description: "Developer tool for inspecting and debugging the Neo4j knowledge graph",
	}, handler.handle)
	return nil
}

func (h *graphToolHandler) handle(ctx context.Context, req *sdkmcp.CallToolRequest, params *GraphToolParams) (*sdkmcp.CallToolResult, any, error) {
	if h.client == nil {
		return textResult("graph_tool unavailable: Neo4j client not configured"), nil, fmt.Errorf("Neo4j client not configured")
	}

	if params == nil {
		return textResult("graph_tool requires parameters"), nil, fmt.Errorf("missing parameters")
	}

	var query string
	var queryParams map[string]interface{}

	if params.Cypher != "" {
		query = params.Cypher
		hasParams := false
		queryParams = make(map[string]interface{})
		if params.JobID != "" {
			queryParams["jobId"] = params.JobID
			hasParams = true
		}
		if params.UserID != "" {
			queryParams["userId"] = params.UserID
			hasParams = true
		}
		if params.Filters != nil && len(params.Filters) > 0 {
			for k, v := range params.Filters {
				queryParams[k] = v
			}
			hasParams = true
		}
		if !hasParams {
			queryParams = nil
		}
	} else if params.JobID != "" {
		query = `
			MATCH (j:Job {id: $jobId})
			OPTIONAL MATCH (j)-[:WORKED_AT]->(c:Company)
			OPTIONAL MATCH (j)-[:REQUIRES]->(s:Skill)
			OPTIONAL MATCH (j)-[hk:HAS_KEYWORD]->(k:Keyword)
			RETURN j, c,
			       collect(DISTINCT s) as skills,
			       collect(DISTINCT {value: k.value, source: hk.source}) as keywords
		`
		queryParams = map[string]interface{}{"jobId": params.JobID}
	} else {
		query = "MATCH (n) RETURN labels(n) as labels, count(n) as count ORDER BY count DESC LIMIT 20"
		queryParams = nil
	}

	result, err := h.executeQuery(ctx, query, queryParams)
	if err != nil {
		return textResult(fmt.Sprintf("graph_tool error: %v", err)), nil, err
	}

	return textResult(result), nil, nil
}

func (h *graphToolHandler) executeQuery(ctx context.Context, query string, params map[string]interface{}) (string, error) {
	session := h.client.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	var allRecords []*neo4j.Record
	var keys []string

	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		for result.Next(ctx) {
			record := result.Record()
			if keys == nil {
				keys = record.Keys
			}
			allRecords = append(allRecords, record)
		}

		if err := result.Err(); err != nil {
			return nil, err
		}

		return nil, nil
	})
	if err != nil {
		return "", fmt.Errorf("query execution failed: %w", err)
	}

	return h.formatCollectedResults(allRecords, keys)
}

func (h *graphToolHandler) formatCollectedResults(records []*neo4j.Record, keys []string) (string, error) {
	if len(records) == 0 {
		return "Query executed successfully but returned no rows", nil
	}

	var sb strings.Builder
	sb.WriteString("Results:\n")
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	for i, record := range records {
		sb.WriteString(fmt.Sprintf("Row %d:\n", i+1))

		for _, key := range keys {
			val, ok := record.Get(key)
			if !ok {
				sb.WriteString(fmt.Sprintf("  %s: <not found>\n", key))
				continue
			}
			formatted := h.formatValue(val)
			sb.WriteString(fmt.Sprintf("  %s: %s\n", key, formatted))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func (h *graphToolHandler) formatValue(val interface{}) string {
	if val == nil {
		return "null"
	}

	switch v := val.(type) {
	case neo4j.Node:
		propsJSON, _ := json.Marshal(v.Props)
		return fmt.Sprintf("Node[%v] %s", v.Labels, string(propsJSON))
	case neo4j.Relationship:
		propsJSON, _ := json.Marshal(v.Props)
		return fmt.Sprintf("Relationship[%s] %s", v.Type, string(propsJSON))
	case []interface{}:
		if len(v) == 0 {
			return "[]"
		}
		items := make([]string, 0, len(v))
		for _, item := range v {
			items = append(items, h.formatValue(item))
		}
		return fmt.Sprintf("[%s]", strings.Join(items, ", "))
	case map[string]interface{}:
		jsonBytes, _ := json.Marshal(v)
		return string(jsonBytes)
	case string:
		return fmt.Sprintf("%q", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%.2f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(jsonBytes)
	}
}
