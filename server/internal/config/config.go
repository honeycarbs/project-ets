package config

import "os"

type Config struct {
	LogLevel string
	// TODO: Neo4jURI, OpenAIKey, SheetsCredsPath, etc.
}

func Load() (Config, error) {
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "info"
	}

	return Config{
		LogLevel: level,
	}, nil
}
