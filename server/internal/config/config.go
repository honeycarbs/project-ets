package config

import "os"

// Config contains runtime settings for the MCP server
type Config struct {
	LogLevel string
	Host     string // default "0.0.0.0"
	Port     string // default os.Getenv("PORT") or "8080"
	// TODO: Neo4jURI, OpenAIKey, SheetsCredsPath, etc.
}

// Load populates configuration from environment variables, seeding defaults first
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

	return cfg, nil
}
