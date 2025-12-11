package adzuna

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestSearchJobsIntegration(t *testing.T) {
	appID := os.Getenv("ADZUNA_APP_ID")
	appKey := os.Getenv("ADZUNA_APP_KEY")
	country := os.Getenv("ADZUNA_COUNTRY")
	if country == "" {
		country = "us"
	}

	if appID == "" || appKey == "" {
		t.Skip("ADZUNA_APP_ID and ADZUNA_APP_KEY must be set to run this test")
	}

	client, err := NewClient(Config{
		AppID:   appID,
		AppKey:  appKey,
		Country: country,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	jobs, err := client.SearchJobs(ctx, "software engineer", SearchParams{
		Location: "Oregon",
	})
	if err != nil {
		t.Fatalf("SearchJobs: %v", err)
	}

	if len(jobs) == 0 {
		t.Log("Adzuna search returned zero jobs; check query or credentials")
		return
	}

	for i, job := range jobs {
		if i >= 5 {
			break
		}
		t.Logf("Result %d: %s @ %s (%s)", i+1, job.Title, job.CompanyName, job.Location)
	}
	t.Logf("Adzuna search returned %d jobs", len(jobs))
}
