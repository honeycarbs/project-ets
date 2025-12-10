package neo4j

import (
	"context"
	"fmt"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Client wraps the Neo4j driver for reuse across repositories
type Client struct {
	driver neo4j.DriverWithContext
}

// Config holds Neo4j connection configuration
type Config struct {
	URI      string
	Username string
	Password string
}

// NewClient creates and verifies a Neo4j client connection
func NewClient(cfg Config) (*Client, error) {
	driver, err := neo4j.NewDriverWithContext(
		cfg.URI,
		neo4j.BasicAuth(cfg.Username, cfg.Password, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	// Verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := driver.VerifyConnectivity(ctx); err != nil {
		_ = driver.Close(ctx)
		return nil, fmt.Errorf("failed to verify Neo4j connectivity: %w", err)
	}

	return &Client{driver: driver}, nil
}

// Driver returns the underlying Neo4j driver for repository use
func (c *Client) Driver() neo4j.DriverWithContext {
	return c.driver
}

// Close closes the Neo4j driver connection
func (c *Client) Close(ctx context.Context) error {
	if c.driver != nil {
		return c.driver.Close(ctx)
	}
	return nil
}

// NewSession creates a new Neo4j session with the given configuration
func (c *Client) NewSession(ctx context.Context, config neo4j.SessionConfig) neo4j.SessionWithContext {
	return c.driver.NewSession(ctx, config)
}

