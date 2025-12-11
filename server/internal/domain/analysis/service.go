package analysis

import (
	"context"
	"time"

	"github.com/honeycarbs/project-ets/internal/mcp/tools"
	"github.com/honeycarbs/project-ets/internal/repository"
)

// Service retrieves graph context for job analysis
type Service struct {
	repo repository.AnalysisRepository
}

// NewService creates an analysis service
func NewService(repo repository.AnalysisRepository) *Service {
	return &Service{repo: repo}
}

// Analyze retrieves job subgraphs and related context from the graph
func (s *Service) Analyze(ctx context.Context, params tools.JobAnalysisParams) (tools.JobAnalysisResult, error) {
	subgraphs, err := s.repo.GetJobSubgraphs(ctx, params.JobIDs)
	if err != nil {
		return tools.JobAnalysisResult{}, err
	}

	summaries := make([]tools.JobAnalysisSummary, 0, len(subgraphs))
	for _, sg := range subgraphs {
		summaries = append(summaries, s.buildSummary(sg, params.Profile, params.Focus))
	}

	return tools.JobAnalysisResult{
		Jobs:        summaries,
		GeneratedAt: time.Now().UTC(),
	}, nil
}

func (s *Service) buildSummary(sg repository.JobSubgraph, profile, focus string) tools.JobAnalysisSummary {
	skills := make([]string, 0, len(sg.Job.Skills))
	for _, skill := range sg.Job.Skills {
		skills = append(skills, skill.Name)
	}

	keywords := make([]tools.KeywordEntry, 0, len(sg.Keywords))
	for _, kw := range sg.Keywords {
		keywords = append(keywords, tools.KeywordEntry{
			Value: kw.Value,
			Notes: kw.Source,
		})
	}

	return tools.JobAnalysisSummary{
		JobID:              sg.Job.ID.String(),
		Summary:            sg.Job.Title + " at " + sg.Job.Company.Name,
		RecommendedKeywords: keywords,
		SupportingData: map[string]any{
			"title":       sg.Job.Title,
			"company":     sg.Job.Company.Name,
			"location":    sg.Job.Location,
			"remote":      sg.Job.Remote,
			"url":         sg.Job.URL,
			"description": sg.Job.Description,
			"skills":      skills,
			"profile":     profile,
			"focus":       focus,
		},
	}
}
