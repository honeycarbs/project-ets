package neo4j

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/honeycarbs/project-ets/internal/domain"
	"github.com/honeycarbs/project-ets/internal/repository"
	"github.com/honeycarbs/project-ets/pkg/logging"
	pkgneo4j "github.com/honeycarbs/project-ets/pkg/neo4j"
)

var _ repository.AnalysisRepository = (*AnalysisRepository)(nil)

// AnalysisRepository implements graph retrieval for job analysis
type AnalysisRepository struct {
	client *pkgneo4j.Client
	logger *logging.Logger
}

// NewAnalysisRepository creates an analysis repository
func NewAnalysisRepository(client *pkgneo4j.Client, logger *logging.Logger) *AnalysisRepository {
	return &AnalysisRepository{
		client: client,
		logger: logger,
	}
}

// GetJobSubgraphs retrieves jobs with their skills and keywords
func (r *AnalysisRepository) GetJobSubgraphs(ctx context.Context, jobIDs []string) ([]repository.JobSubgraph, error) {
	if len(jobIDs) == 0 {
		return nil, nil
	}

	sessionConfig := neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeRead,
		// DatabaseName: "", // empty means default database (usually "neo4j")
	}
	session := r.client.NewSession(ctx, sessionConfig)
	defer session.Close(ctx)

	// First, let's verify we can see ANY jobs at all
	countQuery := "MATCH (j:Job) RETURN count(j) as total"
	totalJobs, countErr := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, countQuery, nil)
		if err != nil {
			return 0, err
		}
		if res.Next(ctx) {
			if total, found := res.Record().Get("total"); found {
				return total, nil
			}
		}
		return 0, res.Err()
	})
	if countErr == nil {
		r.logger.Info("AnalysisRepository.GetJobSubgraphs: total jobs in database",
			"total_jobs", totalJobs,
			"database", sessionConfig.DatabaseName,
		)
	}

	query := `
		MATCH (j:Job)
		WHERE j.id IN $ids
		OPTIONAL MATCH (j)-[:WORKED_AT]->(c:Company)
		OPTIONAL MATCH (j)-[:REQUIRES]->(s:Skill)
		OPTIONAL MATCH (j)-[hk:HAS_KEYWORD]->(k:Keyword)
		RETURN j, c,
		       collect(DISTINCT s) as skills,
		       collect(DISTINCT {value: k.value, source: hk.source}) as keywords
	`

	params := map[string]interface{}{"ids": jobIDs}

	r.logger.Info("AnalysisRepository.GetJobSubgraphs: executing Neo4j query",
		"job_ids", jobIDs,
		"ids_count", len(jobIDs),
		"database", sessionConfig.DatabaseName,
		"query", query,
		"params", params,
	)

	// Collect all records INSIDE the transaction
	records, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, query, params)
		if err != nil {
			r.logger.Error("AnalysisRepository.GetJobSubgraphs: tx.Run failed", "err", err)
			return nil, err
		}
		r.logger.Debug("AnalysisRepository.GetJobSubgraphs: tx.Run succeeded, collecting records")

		// Collect all records into a slice while still in transaction
		var allRecords []*neo4j.Record
		for res.Next(ctx) {
			allRecords = append(allRecords, res.Record())
		}

		if err := res.Err(); err != nil {
			r.logger.Error("AnalysisRepository.GetJobSubgraphs: error iterating results", "err", err)
			return nil, err
		}

		r.logger.Info("AnalysisRepository.GetJobSubgraphs: collected records from Neo4j",
			"records_count", len(allRecords),
		)

		return allRecords, nil
	})
	if err != nil {
		r.logger.Error("AnalysisRepository.GetJobSubgraphs: Neo4j query failed", "err", err)
		return nil, err
	}

	r.logger.Debug("AnalysisRepository.GetJobSubgraphs: ExecuteRead completed, parsing results")

	allRecords := records.([]*neo4j.Record)
	subgraphs, err := r.parseSubgraphRecords(ctx, allRecords)
	if err != nil {
		r.logger.Error("AnalysisRepository.GetJobSubgraphs: failed to parse Neo4j results", "err", err)
		return nil, err
	}

	r.logger.Debug("AnalysisRepository.GetJobSubgraphs: Neo4j query completed",
		"requested_ids", jobIDs,
		"subgraphs_count", len(subgraphs),
	)

	return subgraphs, nil
}

// FindRelatedJobs finds jobs connected via shared skills
func (r *AnalysisRepository) FindRelatedJobs(ctx context.Context, jobID string, limit int) ([]repository.RelatedJob, error) {
	session := r.client.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := `
		MATCH (j:Job {id: $jobId})-[:REQUIRES]->(s:Skill)<-[:REQUIRES]-(related:Job)
		WHERE related.id <> $jobId
		WITH related, collect(DISTINCT s.name) as sharedSkills
		OPTIONAL MATCH (j:Job {id: $jobId})-[:HAS_KEYWORD]->(k:Keyword)<-[:HAS_KEYWORD]-(related)
		WITH related, sharedSkills, collect(DISTINCT k.value) as sharedKeywords
		RETURN related, sharedSkills, sharedKeywords,
		       (size(sharedSkills) * 2 + size(sharedKeywords)) as relevance
		ORDER BY relevance DESC
		LIMIT $limit
	`

	// Collect all records INSIDE the transaction
	records, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, query, map[string]interface{}{
			"jobId": jobID,
			"limit": limit,
		})
		if err != nil {
			return nil, err
		}

		// Collect all records into a slice while still in transaction
		var allRecords []*neo4j.Record
		for res.Next(ctx) {
			allRecords = append(allRecords, res.Record())
		}

		if err := res.Err(); err != nil {
			return nil, err
		}

		return allRecords, nil
	})
	if err != nil {
		return nil, err
	}

	return r.parseRelatedRecords(ctx, records.([]*neo4j.Record))
}

// GetSkillCooccurrences finds skills that commonly appear with given skills
func (r *AnalysisRepository) GetSkillCooccurrences(ctx context.Context, skills []string, limit int) ([]repository.SkillCooccurrence, error) {
	if len(skills) == 0 {
		return nil, nil
	}

	session := r.client.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := `
		MATCH (j:Job)-[:REQUIRES]->(s1:Skill)
		WHERE toLower(s1.name) IN $skills
		MATCH (j)-[:REQUIRES]->(s2:Skill)
		WHERE NOT toLower(s2.name) IN $skills
		WITH s2.name as skill, count(DISTINCT j) as cooccurs, collect(DISTINCT s1.name) as commonWith
		RETURN skill, cooccurs, commonWith
		ORDER BY cooccurs DESC
		LIMIT $limit
	`

	// Collect all records INSIDE the transaction
	records, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, query, map[string]interface{}{
			"skills": skills,
			"limit":  limit,
		})
		if err != nil {
			return nil, err
		}

		// Collect all records into a slice while still in transaction
		var allRecords []*neo4j.Record
		for res.Next(ctx) {
			allRecords = append(allRecords, res.Record())
		}

		if err := res.Err(); err != nil {
			return nil, err
		}

		return allRecords, nil
	})
	if err != nil {
		return nil, err
	}

	return r.parseCooccurrenceRecords(ctx, records.([]*neo4j.Record))
}

func (r *AnalysisRepository) parseSubgraphRecords(ctx context.Context, records []*neo4j.Record) ([]repository.JobSubgraph, error) {
	subgraphs := make([]repository.JobSubgraph, 0, len(records))

	r.logger.Debug("AnalysisRepository.parseSubgraphRecords: starting to parse Neo4j records",
		"records_count", len(records),
	)

	for i, record := range records {
		recordCount := i + 1

		r.logger.Debug("AnalysisRepository.parseSubgraphRecords: processing record",
			"record_index", recordCount,
			"record_keys", record.Keys,
		)

		// Log the raw job node data
		if jobVal, ok := record.Get("j"); ok {
			r.logger.Debug("AnalysisRepository.parseSubgraphRecords: raw job node",
				"record_index", recordCount,
				"job_val_type", fmt.Sprintf("%T", jobVal),
				"job_val", jobVal,
			)
		} else {
			r.logger.Warn("AnalysisRepository.parseSubgraphRecords: record missing 'j' key",
				"record_index", recordCount,
			)
		}

		job, err := r.parseJobNode(record)
		if err != nil {
			r.logger.Error("AnalysisRepository.parseSubgraphRecords: failed to parse job node",
				"record_index", recordCount,
				"err", err,
			)
			return nil, fmt.Errorf("failed to parse job node at record %d: %w", recordCount, err)
		}

		// Check if we got an empty job (no error but also no valid job)
		if job.ID == uuid.Nil {
			r.logger.Warn("AnalysisRepository.parseSubgraphRecords: parsed job has nil ID, skipping",
				"record_index", recordCount,
			)
			continue
		}

		r.logger.Debug("AnalysisRepository.parseSubgraphRecords: successfully parsed job",
			"record_index", recordCount,
			"job_id", job.ID.String(),
			"job_title", job.Title,
		)

		job.Company = r.parseCompanyNode(record)
		job.Skills = r.parseSkillNodes(record)

		keywords := r.parseKeywordNodes(record)

		r.logger.Debug("AnalysisRepository.parseSubgraphRecords: parsed job details",
			"record_index", recordCount,
			"job_id", job.ID.String(),
			"company", job.Company.Name,
			"skills_count", len(job.Skills),
			"keywords_count", len(keywords),
		)

		subgraphs = append(subgraphs, repository.JobSubgraph{
			Job:      job,
			Keywords: keywords,
		})
	}

	r.logger.Info("AnalysisRepository.parseSubgraphRecords: completed parsing",
		"total_records", len(records),
		"subgraphs_created", len(subgraphs),
	)

	return subgraphs, nil
}

func (r *AnalysisRepository) parseJobNode(record *neo4j.Record) (domain.Job, error) {
	jobVal, ok := record.Get("j")
	if !ok {
		r.logger.Debug("AnalysisRepository.parseJobNode: record does not contain 'j' key")
		return domain.Job{}, nil
	}

	jobNode, ok := jobVal.(neo4j.Node)
	if !ok {
		r.logger.Warn("AnalysisRepository.parseJobNode: 'j' value is not a neo4j.Node",
			"actual_type", fmt.Sprintf("%T", jobVal),
			"value", jobVal,
		)
		return domain.Job{}, nil
	}

	props := jobNode.Props
	r.logger.Debug("AnalysisRepository.parseJobNode: extracted job node properties",
		"props", props,
		"props_count", len(props),
	)

	rawID := getStringProp(props, "id")
	r.logger.Debug("AnalysisRepository.parseJobNode: attempting to parse job ID",
		"raw_id", rawID,
		"raw_id_length", len(rawID),
	)

	jobID, err := uuid.Parse(rawID)
	if err != nil {
		r.logger.Error("AnalysisRepository.parseJobNode: UUID parse failed",
			"raw_id", rawID,
			"err", err,
			"all_props", props,
		)
		return domain.Job{}, fmt.Errorf("parse job id %q: %w (props=%v)", rawID, err, props)
	}

	r.logger.Debug("AnalysisRepository.parseJobNode: successfully parsed job ID",
		"job_id", jobID.String(),
		"title", getStringProp(props, "title"),
	)

	return domain.Job{
		ID:          jobID,
		Title:       getStringProp(props, "title"),
		Location:    getStringProp(props, "location"),
		Remote:      getBoolProp(props, "remote"),
		URL:         getStringProp(props, "url"),
		Source:      getStringProp(props, "source"),
		ExternalID:  getStringProp(props, "externalId"),
		PostedAt:    getTimeProp(props, "postedAt"),
		Description: getStringProp(props, "description"),
		Score:       getFloatProp(props, "score"),
		FetchedAt:   getTimeProp(props, "fetchedAt"),
	}, nil
}

func (r *AnalysisRepository) parseCompanyNode(record *neo4j.Record) domain.CompanyRef {
	companyVal, ok := record.Get("c")
	if !ok || companyVal == nil {
		return domain.CompanyRef{}
	}

	companyNode, ok := companyVal.(neo4j.Node)
	if !ok {
		return domain.CompanyRef{}
	}

	return domain.CompanyRef{
		ID:   getStringProp(companyNode.Props, "id"),
		Name: getStringProp(companyNode.Props, "name"),
	}
}

func (r *AnalysisRepository) parseSkillNodes(record *neo4j.Record) []domain.SkillRef {
	skillsVal, ok := record.Get("skills")
	if !ok {
		return nil
	}

	skillsList, ok := skillsVal.([]interface{})
	if !ok {
		return nil
	}

	skills := make([]domain.SkillRef, 0, len(skillsList))
	for _, sv := range skillsList {
		if skillNode, ok := sv.(neo4j.Node); ok {
			skills = append(skills, domain.SkillRef{
				ID:   getStringProp(skillNode.Props, "id"),
				Name: getStringProp(skillNode.Props, "name"),
			})
		}
	}
	return skills
}

func (r *AnalysisRepository) parseKeywordNodes(record *neo4j.Record) []repository.KeywordNode {
	keywordsVal, ok := record.Get("keywords")
	if !ok {
		return nil
	}

	keywordsList, ok := keywordsVal.([]interface{})
	if !ok {
		return nil
	}

	keywords := make([]repository.KeywordNode, 0, len(keywordsList))
	for _, kv := range keywordsList {
		if kwMap, ok := kv.(map[string]interface{}); ok {
			value := getStringFromMap(kwMap, "value")
			if value == "" {
				continue
			}
			keywords = append(keywords, repository.KeywordNode{
				Value:  value,
				Source: getStringFromMap(kwMap, "source"),
			})
		}
	}
	return keywords
}

func (r *AnalysisRepository) parseRelatedRecords(ctx context.Context, records []*neo4j.Record) ([]repository.RelatedJob, error) {
	related := make([]repository.RelatedJob, 0, len(records))

	for _, record := range records {
		job, err := r.parseJobNode(record)
		if err != nil {
			continue
		}

		sharedSkills := getStringSlice(record, "sharedSkills")
		sharedKeywords := getStringSlice(record, "sharedKeywords")
		relevance := getRecordFloat(record, "relevance")

		related = append(related, repository.RelatedJob{
			Job:            job,
			SharedSkills:   sharedSkills,
			SharedKeywords: sharedKeywords,
			Relevance:      relevance,
		})
	}

	return related, nil
}

func (r *AnalysisRepository) parseCooccurrenceRecords(ctx context.Context, records []*neo4j.Record) ([]repository.SkillCooccurrence, error) {
	cooccurrences := make([]repository.SkillCooccurrence, 0, len(records))

	for _, record := range records {
		skill, _ := record.Get("skill")
		cooccurs, _ := record.Get("cooccurs")
		commonWith := getStringSlice(record, "commonWith")

		cooccurrences = append(cooccurrences, repository.SkillCooccurrence{
			Skill:      skill.(string),
			Cooccurs:   int(cooccurs.(int64)),
			CommonWith: commonWith,
		})
	}

	return cooccurrences, nil
}

func getStringProp(props map[string]interface{}, key string) string {
	if v, ok := props[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBoolProp(props map[string]interface{}, key string) bool {
	if v, ok := props[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getFloatProp(props map[string]interface{}, key string) float64 {
	if v, ok := props[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

func getTimeProp(props map[string]interface{}, key string) time.Time {
	if v, ok := props[key]; ok {
		if t, ok := v.(time.Time); ok {
			return t
		}
		if dt, ok := v.(neo4j.LocalDateTime); ok {
			return dt.Time()
		}
	}
	return time.Time{}
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getStringSlice(record *neo4j.Record, key string) []string {
	val, ok := record.Get(key)
	if !ok || val == nil {
		return nil
	}

	list, ok := val.([]interface{})
	if !ok {
		return nil
	}

	result := make([]string, 0, len(list))
	for _, v := range list {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func getRecordFloat(record *neo4j.Record, key string) float64 {
	val, ok := record.Get(key)
	if !ok || val == nil {
		return 0
	}
	if f, ok := val.(float64); ok {
		return f
	}
	if i, ok := val.(int64); ok {
		return float64(i)
	}
	return 0
}
