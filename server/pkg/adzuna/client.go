package adzuna

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	defaultBaseURL  = "https://api.adzuna.com"
	defaultCountry  = "us"
	defaultPageSize = 20
)

// NewClient instantiates an Adzuna API client
func NewClient(cfg Config) (*Client, error) {
	if cfg.AppID == "" || cfg.AppKey == "" {
		return nil, fmt.Errorf("adzuna: app_id and app_key are required")
	}

	country := cfg.Country
	if country == "" {
		country = defaultCountry
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	pageSize := cfg.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}

	return &Client{
		appID:      cfg.AppID,
		appKey:     cfg.AppKey,
		country:    country,
		baseURL:    baseURL,
		httpClient: httpClient,
		pageSize:   pageSize,
	}, nil
}

// SearchJobs queries Adzuna with keyword/location filters
func (c *Client) SearchJobs(ctx context.Context, query string, params SearchParams) ([]Job, error) {
	if c == nil {
		return nil, fmt.Errorf("adzuna: client is nil")
	}

	u, err := c.buildSearchURL(query, params)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("adzuna: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("adzuna: request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("adzuna: API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload jobSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("adzuna: decode response: %w", err)
	}

	jobs := make([]Job, 0, len(payload.Results))
	for _, posting := range payload.Results {
		job := mapPosting(posting)
		if job.ID == "" {
			job.ID = uuid.NewString()
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (c *Client) buildSearchURL(query string, params SearchParams) (string, error) {
	if query == "" {
		return "", fmt.Errorf("adzuna: query is required")
	}

	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("adzuna: parse base url: %w", err)
	}

	u.Path = path.Join(u.Path, "v1", "api", "jobs", c.country, "search", "1")

	values := url.Values{}
	values.Set("app_id", c.appID)
	values.Set("app_key", c.appKey)
	values.Set("what", query)
	values.Set("results_per_page", fmt.Sprint(c.pageSize))
	values.Set("content-type", "application/json")

	if params.Location != "" {
		values.Set("where", params.Location)
	}

	if params.Remote != nil {
		if *params.Remote {
			values.Set("distance", "0") // remote filter approximation
			values.Set("where", "Remote")
		}
	}

	if len(params.Skills) > 0 {
		values.Set("skills", strings.Join(params.Skills, ","))
	}

	u.RawQuery = values.Encode()
	return u.String(), nil
}

func mapPosting(posting jobPosting) Job {
	job := Job{
		ID:          posting.ID,
		Title:       posting.Title,
		CompanyName: posting.Company.DisplayName,
		Location:    posting.Location.DisplayName,
		URL:         posting.RedirectURL,
		Description: posting.Description,
		SalaryMin:   posting.SalaryMin,
		SalaryMax:   posting.SalaryMax,
		FetchedAt:   time.Now().UTC(),
	}

	if posting.Created != "" {
		if ts, err := time.Parse(time.RFC3339, posting.Created); err == nil {
			job.PostedAt = ts
		}
	}

	if strings.EqualFold(posting.Contract, "remote") {
		job.Remote = true
	}

	return job
}
