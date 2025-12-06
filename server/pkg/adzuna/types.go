package adzuna

import (
	"net/http"
	"time"
)

// Config defines Adzuna API client settings
type Config struct {
	AppID      string
	AppKey     string
	Country    string
	BaseURL    string
	HTTPClient *http.Client
	PageSize   int
}

// Client queries Adzuna job search API
type Client struct {
	appID      string
	appKey     string
	country    string
	baseURL    string
	httpClient *http.Client
	pageSize   int
}

// SearchParams describe a job search request
type SearchParams struct {
	Location string
	Remote   *bool
	Skills   []string
}

type jobSearchResponse struct {
	Count   int          `json:"count"`
	Results []jobPosting `json:"results"`
	Mean    float64      `json:"mean"`
	Median  float64      `json:"median"`
	Pages   int          `json:"pages"`
}

type jobPosting struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Company     companySummary  `json:"company"`
	Location    locationSummary `json:"location"`
	Description string          `json:"description"`
	Created     string          `json:"created"`
	RedirectURL string          `json:"redirect_url"`
	Contract    string          `json:"contract_time"`
	Category    struct {
		Label string `json:"label"`
	} `json:"category"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	SalaryMin float64 `json:"salary_min"`
	SalaryMax float64 `json:"salary_max"`
}

type companySummary struct {
	DisplayName string `json:"display_name"`
}

type locationSummary struct {
	DisplayName string `json:"display_name"`
}

// Job represents a normalized Adzuna job posting.
type Job struct {
	ID          string
	Title       string
	CompanyName string
	Location    string
	URL         string
	Description string
	Remote      bool
	PostedAt    time.Time
	SalaryMin   float64
	SalaryMax   float64
	FetchedAt   time.Time
}
