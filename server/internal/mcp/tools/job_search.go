package tools

import (
	"context"
	"encoding/json"
)

type JobSearch struct{}

var (
	jobSearchInputSchema = json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": { "type": "string", "description": "Natural language search query" },
			"filters": {
				"type": "object",
				"properties": {
					"location": { "type": "string" },
					"remote": { "type": "boolean" },
					"skills": {
						"type": "array",
						"items": { "type": "string" }
					}
				}
			}
		},
		"required": ["query"]
	}`)

	jobSearchResultSchema = json.RawMessage(`{
		"type": "object",
		"properties": {
			"jobs": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"id": { "type": "string" },
						"title": { "type": "string" },
						"company": { "type": "string" },
						"score": { "type": "number" },
						"source": { "type": "string" }
					}
				}
			},
			"meta": {
				"type": "object",
				"properties": {
					"fetched_at": { "type": "string", "format": "date-time" },
					"source_count": { "type": "integer" }
				}
			}
		}
	}`)
)

func NewJobSearch() *JobSearch {
	return &JobSearch{}
}

func (t *JobSearch) Name() string {
	return "job_search"
}

func (t *JobSearch) Description() string {
	return "Searches external job boards/APIs, normalizes, and stores job postings"
}

func (t *JobSearch) InputSchema() json.RawMessage {
	return jobSearchInputSchema
}

func (t *JobSearch) ResultSchema() json.RawMessage {
	return jobSearchResultSchema
}

func (t *JobSearch) Execute(_ context.Context, params json.RawMessage) (any, error) {
	return stubResponse(
		t.Name(),
		"Stub implementation: fetch from LinkedIn/Indeed, normalize, and persist to Neo4j",
	), nil
}
