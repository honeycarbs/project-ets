package neo4j

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/honeycarbs/project-ets/internal/mcp/tools"
	pkgneo4j "github.com/honeycarbs/project-ets/pkg/neo4j"
)

// KeywordRepository implements tools.KeywordRepository with Neo4j
type KeywordRepository struct {
	client *pkgneo4j.Client
}

// NewKeywordRepository creates a KeywordRepository with a Neo4j client
func NewKeywordRepository(client *pkgneo4j.Client) *KeywordRepository {
	return &KeywordRepository{
		client: client,
	}
}

// PersistKeywords stores keyword records in Neo4j, linking them to existing Job nodes
func (r *KeywordRepository) PersistKeywords(ctx context.Context, records []tools.KeywordRecord) error {
	if len(records) == 0 {
		return nil
	}

	session := r.client.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	query := `
		UNWIND $records AS record
		MATCH (j:Job {id: record.jobId})
		WITH j, record
		UNWIND record.keywords AS keyword
		MERGE (k:Keyword {value: keyword.value})
		SET k.notes = coalesce(CASE WHEN keyword.notes <> "" THEN keyword.notes ELSE null END, k.notes)
		MERGE (j)-[rel:HAS_KEYWORD]->(k)
		SET rel.createdAt = coalesce(rel.createdAt, datetime()),
		    rel.source = coalesce(CASE WHEN record.source <> "" THEN record.source ELSE null END, rel.source)
	`

	recordsData := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		keywordsData := make([]map[string]interface{}, 0, len(record.Keywords))
		for _, keyword := range record.Keywords {
			keywordData := map[string]interface{}{
				"value": keyword.Value,
			}
			if keyword.Notes != "" {
				keywordData["notes"] = keyword.Notes
			}
			keywordsData = append(keywordsData, keywordData)
		}

		recordData := map[string]interface{}{
			"jobId":    record.JobID,
			"keywords": keywordsData,
		}
		if record.Source != "" {
			recordData["source"] = record.Source
		}
		recordsData = append(recordsData, recordData)
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, map[string]interface{}{"records": recordsData})
		if err != nil {
			return nil, fmt.Errorf("failed to execute keyword persistence query: %w", err)
		}
		return result.Consume(ctx)
	})

	return err
}

