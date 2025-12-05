package domain

import (
	"time"

	"github.com/google/uuid"
)

// JobID is the internal identifier for a Job (UUID)
type JobID = uuid.UUID

// CompanyRef is a reference to a company
type CompanyRef struct {
	ID   string
	Name string
}

// SkillRef is a reference to a skill
type SkillRef struct {
	ID   string
	Name string
}

// Job is the core domain entity representing a normalized job posting
type Job struct {
	ID          JobID
	Title       string
	Company     CompanyRef
	Location    string
	Remote      bool
	URL         string
	Source      string
	ExternalID  string
	PostedAt    time.Time
	Description string
	Skills      []SkillRef
	Score       float64
	FetchedAt   time.Time
}

// JobSearchFilters represent the filters the user can apply in job_search
type JobSearchFilters struct {
	Location string
	Remote   *bool
	Skills   []string
}

// JobSummary is a view of a job for responses (MCP, Sheets, etc.)
type JobSummary struct {
	ID       JobID   `json:"id"`
	Title    string  `json:"title"`
	Company  string  `json:"company"`
	Location string  `json:"location"`
	Remote   bool    `json:"remote"`
	URL      string  `json:"url"`
	Source   string  `json:"source"`
	Score    float64 `json:"score"`
}

// JobSearchResult is the domain result for a job search
type JobSearchResult struct {
	Jobs        []JobSummary
	FetchedAt   time.Time
	SourceCount int
}
