package tools

import (
	"context"
	"encoding/json"
)

type JobAnalysis struct{}

var (
	jobAnalysisInputSchema = json.RawMessage(`{
		"type": "object",
		"properties": {
			"job_ids": {
				"type": "array",
				"items": { "type": "string" },
				"description": "Job identifiers stored in Neo4j"
			},
			"profile": {
				"type": "string",
				"description": "Free-form resume/profile to compare against"
			},
			"focus": {
				"type": "string",
				"description": "Optional prompt such as 'compare to my resume'"
			}
		},
		"oneOf": [
			{ "required": ["job_ids"] },
			{ "required": ["profile"] }
		]
	}`)

	jobAnalysisResultSchema = json.RawMessage(`{
		"type": "object",
		"properties": {
			"insights": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"bullet": { "type": "string" },
						"evidence": { "type": "array", "items": { "type": "string" } }
					},
					"required": ["bullet"]
				}
			},
			"references": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"job_id": { "type": "string" },
						"nodes": {
							"type": "array",
							"items": { "type": "string" }
						}
					}
				}
			}
		}
	}`)
)

func NewJobAnalysis() *JobAnalysis {
	return &JobAnalysis{}
}

func (t *JobAnalysis) Name() string {
	return "job_analysis"
}

func (t *JobAnalysis) Description() string {
	return "Summarizes stored job graphs against a candidate profile using Graph RAG"
}

func (t *JobAnalysis) InputSchema() json.RawMessage {
	return jobAnalysisInputSchema
}

func (t *JobAnalysis) ResultSchema() json.RawMessage {
	return jobAnalysisResultSchema
}

func (t *JobAnalysis) Execute(_ context.Context, params json.RawMessage) (any, error) {
	return stubResponse(
		t.Name(),
		"Stub implementation: hydrate Graph RAG context and call LLM for insights",
	), nil
}
