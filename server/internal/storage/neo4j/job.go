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
		OPTIONAL MATCH (j)-[:REQUIRES]->(s:Skill)
		RETURN j, collect(DISTINCT c) as companies, collect(DISTINCT s) as skills
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, query, map[string]interface{}{"ids": idStrings})
	})
	if err != nil {
		return nil, err
	}

	records := result.(neo4j.ResultWithContext)
	jobs := make([]domain.Job, 0)

	for records.Next(ctx) {
		record := records.Record()

		jobVal, ok := record.Get("j")
		if !ok {
			continue
		}
		jobNode, ok := jobVal.(neo4j.Node)
		if !ok {
			continue
		}

		props := jobNode.Props
		jobID, err := uuid.Parse(props["id"].(string))
		if err != nil {
			continue
		}

		var company domain.CompanyRef
		companiesVal, ok := record.Get("companies")
		if ok {
			if companiesList, ok := companiesVal.([]interface{}); ok && len(companiesList) > 0 {
				if companyNode, ok := companiesList[0].(neo4j.Node); ok {
					companyProps := companyNode.Props
					company = domain.CompanyRef{
						ID:   companyProps["id"].(string),
						Name: companyProps["name"].(string),
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
							ID:   skillProps["id"].(string),
							Name: skillProps["name"].(string),
						})
					}
				}
			}
		}

		var postedAt, fetchedAt time.Time
		if postedAtVal, ok := props["postedAt"]; ok {
			if dt, ok := postedAtVal.(time.Time); ok {
				postedAt = dt
			} else if dt, ok := postedAtVal.(neo4j.LocalDateTime); ok {
				postedAt = dt.Time()
			}
		}
		if fetchedAtVal, ok := props["fetchedAt"]; ok {
			if dt, ok := fetchedAtVal.(time.Time); ok {
				fetchedAt = dt
			} else if dt, ok := fetchedAtVal.(neo4j.LocalDateTime); ok {
				fetchedAt = dt.Time()
			}
		}

		job := domain.Job{
			ID:          jobID,
			Title:       props["title"].(string),
			Company:     company,
			Location:    props["location"].(string),
			Remote:      props["remote"].(bool),
			URL:         props["url"].(string),
			Source:      props["source"].(string),
			ExternalID:  props["externalId"].(string),
			PostedAt:    postedAt,
			Description: props["description"].(string),
			Skills:      jobSkills,
			Score:       props["score"].(float64),
			FetchedAt:   fetchedAt,
		}

		jobs = append(jobs, job)
	}

	if err := records.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

