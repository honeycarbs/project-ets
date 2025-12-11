package neo4j

import (
	"context"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/honeycarbs/project-ets/internal/domain"
	"github.com/honeycarbs/project-ets/internal/repository"

	pkgneo4j "github.com/honeycarbs/project-ets/pkg/neo4j"
)

// Ensure JobRepository implements repository.JobRepository
var _ repository.JobRepository = (*JobRepository)(nil)

// JobRepository implements repository.JobRepository with Neo4j
type JobRepository struct {
	client *pkgneo4j.Client
}

// NewJobRepository creates a JobRepository with a Neo4j client
func NewJobRepository(client *pkgneo4j.Client) *JobRepository {
	return &JobRepository{
		client: client,
	}
}

// UpsertJobs will merge and set job data in Neo4j
func (r *JobRepository) UpsertJobs(ctx context.Context, jobs []domain.Job) error {
	if len(jobs) == 0 {
		return nil
	}

	session := r.client.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	query := `
		UNWIND $jobs AS job
		MERGE (j:Job {source: job.source, externalId: job.externalId})
		SET j.id = job.id,
		    j.title = job.title,
		    j.location = job.location,
		    j.remote = job.remote,
		    j.url = job.url,
		    j.postedAt = datetime({epochMillis: job.postedAt}),
		    j.description = job.description,
		    j.score = job.score,
		    j.fetchedAt = datetime({epochMillis: job.fetchedAt})
		WITH j, job
		MERGE (c:Company {id: job.company.id})
		SET c.name = job.company.name
		MERGE (j)-[:WORKED_AT]->(c)
		WITH j, job
		FOREACH (skill IN job.skills |
			MERGE (s:Skill {id: skill.id})
			SET s.name = skill.name
			MERGE (j)-[:REQUIRES]->(s)
		)
	`

	jobsData := make([]map[string]interface{}, 0, len(jobs))
	for _, job := range jobs {
		skillsData := make([]map[string]interface{}, 0, len(job.Skills))
		for _, skill := range job.Skills {
			skillsData = append(skillsData, map[string]interface{}{
				"id":   skill.ID,
				"name": skill.Name,
			})
		}

		jobsData = append(jobsData, map[string]interface{}{
			"id":          job.ID.String(),
			"title":       job.Title,
			"company":     map[string]interface{}{"id": job.Company.ID, "name": job.Company.Name},
			"location":    job.Location,
			"remote":      job.Remote,
			"url":         job.URL,
			"source":      job.Source,
			"externalId":  job.ExternalID,
			"postedAt":    job.PostedAt.UnixMilli(),
			"description": job.Description,
			"skills":      skillsData,
			"score":       job.Score,
			"fetchedAt":   job.FetchedAt.UnixMilli(),
		})
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, map[string]interface{}{"jobs": jobsData})
		if err != nil {
			return nil, err
		}
		return result.Consume(ctx)
	})

	return err
}

// FindByIDs loads jobs by ID
func (r *JobRepository) FindByIDs(ctx context.Context, ids []domain.JobID) ([]domain.Job, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	session := r.client.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	idStrings := make([]string, 0, len(ids))
	for _, id := range ids {
		idStrings = append(idStrings, id.String())
	}

	query := `
		MATCH (j:Job)
		WHERE j.id IN $ids
		OPTIONAL MATCH (j)-[:WORKED_AT]->(c:Company)
		WITH j, collect(DISTINCT c) as companies
		OPTIONAL MATCH (j)-[:REQUIRES]->(s:Skill)
		WITH j, companies, collect(DISTINCT s) as skills
		RETURN j, companies, skills
	`

	var allRecords []*neo4j.Record
	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, map[string]interface{}{"ids": idStrings})
		if err != nil {
			return nil, err
		}

		for result.Next(ctx) {
			allRecords = append(allRecords, result.Record())
		}

		if err := result.Err(); err != nil {
			return nil, err
		}

		return nil, nil
	})
	if err != nil {
		return nil, err
	}

	jobs := make([]domain.Job, 0)

	for _, record := range allRecords {
		jobVal, ok := record.Get("j")
		if !ok {
			continue
		}
		jobNode, ok := jobVal.(neo4j.Node)
		if !ok {
			continue
		}

		props := jobNode.Props
		jobIDStr := getStringProp(props, "id")
		if jobIDStr == "" {
			continue
		}
		jobID, err := uuid.Parse(jobIDStr)
		if err != nil {
			continue
		}

		var company domain.CompanyRef
		companiesVal, ok := record.Get("companies")
		if ok {
			if companiesList, ok := companiesVal.([]interface{}); ok {
				for _, companyVal := range companiesList {
					if companyNode, ok := companyVal.(neo4j.Node); ok {
						companyProps := companyNode.Props
						company = domain.CompanyRef{
							ID:   getStringProp(companyProps, "id"),
							Name: getStringProp(companyProps, "name"),
						}
						break
					}
				}
			}
		}

		jobSkills := make([]domain.SkillRef, 0)
		skillsVal, ok := record.Get("skills")
		if ok {
			if skillsList, ok := skillsVal.([]interface{}); ok {
				for _, skillVal := range skillsList {
					if skillNode, ok := skillVal.(neo4j.Node); ok {
						skillProps := skillNode.Props
						jobSkills = append(jobSkills, domain.SkillRef{
							ID:   getStringProp(skillProps, "id"),
							Name: getStringProp(skillProps, "name"),
						})
					}
				}
			}
		}

		postedAt := getTimeProp(props, "postedAt")
		fetchedAt := getTimeProp(props, "fetchedAt")

		job := domain.Job{
			ID:          jobID,
			Title:       getStringProp(props, "title"),
			Company:     company,
			Location:    getStringProp(props, "location"),
			Remote:      getBoolProp(props, "remote"),
			URL:         getStringProp(props, "url"),
			Source:      getStringProp(props, "source"),
			ExternalID:  getStringProp(props, "externalId"),
			PostedAt:    postedAt,
			Description: getStringProp(props, "description"),
			Skills:      jobSkills,
			Score:       getFloatProp(props, "score"),
			FetchedAt:   fetchedAt,
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}
