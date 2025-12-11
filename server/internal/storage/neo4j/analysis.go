package neo4j

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/honeycarbs/project-ets/internal/domain"
	"github.com/honeycarbs/project-ets/internal/repository"
	pkgneo4j "github.com/honeycarbs/project-ets/pkg/neo4j"
)

var _ repository.AnalysisRepository = (*AnalysisRepository)(nil)

// AnalysisRepository implements graph retrieval for job analysis
type AnalysisRepository struct {
	client *pkgneo4j.Client
}

// NewAnalysisRepository creates an analysis repository
func NewAnalysisRepository(client *pkgneo4j.Client) *AnalysisRepository {
	return &AnalysisRepository{client: client}
}

// GetJobSubgraphs retrieves jobs with their skills and keywords
func (r *AnalysisRepository) GetJobSubgraphs(ctx context.Context, jobIDs []string) ([]repository.JobSubgraph, error) {
	if len(jobIDs) == 0 {
		return nil, nil
	}

	session := r.client.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

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

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]interface{}{"ids": jobIDs})
	})
	if err != nil {
		return nil, err
	}

	return r.parseSubgraphResults(ctx, result.(neo4j.ResultWithContext))
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

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"jobId": jobID,
			"limit": limit,
		})
	})
	if err != nil {
		return nil, err
	}

	return r.parseRelatedResults(ctx, result.(neo4j.ResultWithContext))
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

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]interface{}{
			"skills": skills,
			"limit":  limit,
		})
	})
	if err != nil {
		return nil, err
	}

	return r.parseCooccurrenceResults(ctx, result.(neo4j.ResultWithContext))
}

func (r *AnalysisRepository) parseSubgraphResults(ctx context.Context, records neo4j.ResultWithContext) ([]repository.JobSubgraph, error) {
	subgraphs := make([]repository.JobSubgraph, 0)

	for records.Next(ctx) {
		record := records.Record()

		job, err := r.parseJobNode(record)
		if err != nil {
			continue
		}

		job.Company = r.parseCompanyNode(record)
		job.Skills = r.parseSkillNodes(record)

		keywords := r.parseKeywordNodes(record)

		subgraphs = append(subgraphs, repository.JobSubgraph{
			Job:      job,
			Keywords: keywords,
		})
	}

	return subgraphs, records.Err()
}

func (r *AnalysisRepository) parseJobNode(record *neo4j.Record) (domain.Job, error) {
	jobVal, ok := record.Get("j")
	if !ok {
		return domain.Job{}, nil
	}

	jobNode, ok := jobVal.(neo4j.Node)
	if !ok {
		return domain.Job{}, nil
	}

	props := jobNode.Props
	jobID, err := uuid.Parse(getStringProp(props, "id"))
	if err != nil {
		return domain.Job{}, err
	}

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

func (r *AnalysisRepository) parseRelatedResults(ctx context.Context, records neo4j.ResultWithContext) ([]repository.RelatedJob, error) {
	related := make([]repository.RelatedJob, 0)

	for records.Next(ctx) {
		record := records.Record()

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

	return related, records.Err()
}

func (r *AnalysisRepository) parseCooccurrenceResults(ctx context.Context, records neo4j.ResultWithContext) ([]repository.SkillCooccurrence, error) {
	cooccurrences := make([]repository.SkillCooccurrence, 0)

	for records.Next(ctx) {
		record := records.Record()

		skill, _ := record.Get("skill")
		cooccurs, _ := record.Get("cooccurs")
		commonWith := getStringSlice(record, "commonWith")

		cooccurrences = append(cooccurrences, repository.SkillCooccurrence{
			Skill:      skill.(string),
			Cooccurs:   int(cooccurs.(int64)),
			CommonWith: commonWith,
		})
	}

	return cooccurrences, records.Err()
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

