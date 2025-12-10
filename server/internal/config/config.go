package config

import (
	"fmt"
	"os"
	"strings"
)

// Config contains runtime settings for the MCP server
type Config struct {
	LogLevel string
	Host     string // default 0.0.0.0
	Port     string // default PORT env or 8080
	Adzuna   struct {
		AppID   string
		AppKey  string
		Country string
	} // Adzuna API credentials
	Neo4j struct {
		URI      string
		Username string
		Password string
	} 
	// TODO: add OpenAIKey, SheetsCredsPath
}

// Load populates config from environment variables
func Load() (Config, error) {
	cfg := Config{
		LogLevel: "info",
		Host:     "0.0.0.0",
		Port:     "8080",
	}

	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	if v := os.Getenv("MCP_HOST"); v != "" {
		cfg.Host = v
	}

	if v := os.Getenv("PORT"); v != "" {
		cfg.Port = v
	}

	cfg.Adzuna.AppID = os.Getenv("ADZUNA_APP_ID")
	cfg.Adzuna.AppKey = os.Getenv("ADZUNA_APP_KEY")
	if v := os.Getenv("ADZUNA_COUNTRY"); v != "" {
		cfg.Adzuna.Country = v
	} else {
		cfg.Adzuna.Country = "us"
	}

	cfg.Neo4j.URI = os.Getenv("NEO4J_URI")
	cfg.Neo4j.Username = os.Getenv("NEO4J_USERNAME")
	cfg.Neo4j.Password = os.Getenv("NEO4J_PASSWORD")

	var missingVars []string

	if cfg.Neo4j.URI == "" {
		missingVars = append(missingVars, "NEO4J_URI")
	}

	if cfg.Neo4j.Username == "" {
		missingVars = append(missingVars, "NEO4J_USERNAME")
	}

	if cfg.Neo4j.Password == "" {
		missingVars = append(missingVars, "NEO4J_PASSWORD")
	}

	if len(missingVars) > 0 {
		return cfg, fmt.Errorf("missing required environment variables: %s", strings.Join(missingVars, ", "))
	}

	return cfg, nil
}
