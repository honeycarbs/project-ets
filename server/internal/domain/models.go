package domain

import (
	"time"

	"github.com/google/uuid"
)

// JobID uniquely identifies a job
type JobID = uuid.UUID

// CompanyRef references a company
type CompanyRef struct {
	ID   string
	Name string
}

// SkillRef references a skill
type SkillRef struct {
	ID   string
	Name string
}

// Job is the normalized job posting entity
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

// JobSearchFilters describe allowed job query filters
type JobSearchFilters struct {
	Location string
	Remote   *bool
	Skills   []string
}

// JobSummary is the response-friendly job view
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

// JobSearchResult wraps job search output
type JobSearchResult struct {
	Jobs        []JobSummary
	FetchedAt   time.Time
	SourceCount int
}
