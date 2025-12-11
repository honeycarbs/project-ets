package repository

import (
	"context"

	"github.com/honeycarbs/project-ets/internal/domain"
)

// JobSubgraph represents a job with its graph neighborhood
type JobSubgraph struct {
	Job      domain.Job
	Keywords []KeywordNode
}

// KeywordNode represents a keyword with its metadata
type KeywordNode struct {
	Value  string
	Source string
}

// RelatedJob represents a job connected via shared graph elements
type RelatedJob struct {
	Job           domain.Job
	SharedSkills  []string
	SharedKeywords []string
	Relevance     float64
}

// SkillCooccurrence represents skills frequently appearing together
type SkillCooccurrence struct {
	Skill       string
	Cooccurs    int
	CommonWith  []string
}

// AnalysisRepository defines graph retrieval operations for job analysis
type AnalysisRepository interface {
	GetJobSubgraphs(ctx context.Context, jobIDs []string) ([]JobSubgraph, error)
	FindRelatedJobs(ctx context.Context, jobID string, limit int) ([]RelatedJob, error)
	GetSkillCooccurrences(ctx context.Context, skills []string, limit int) ([]SkillCooccurrence, error)
}

