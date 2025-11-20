package neo4j

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Neo4jClient struct {
	driver neo4j.DriverWithContext
	ctx    context.Context
}

// NewNeo4jClient creates a new Neo4j client
func NewNeo4jClient(uri, username, password string) (*Neo4jClient, error) {
	ctx := context.Background()

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create driver: %w", err)
	}

	// Verify connectivity
	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to verify connectivity: %w", err)
	}

	return &Neo4jClient{
		driver: driver,
		ctx:    ctx,
	}, nil
}

// Close closes the Neo4j connection
func (c *Neo4jClient) Close() error {
	return c.driver.Close(c.ctx)
}

// ClearDatabase removes all nodes and relationships
func (c *Neo4jClient) ClearDatabase() error {
	session := c.driver.NewSession(c.ctx, neo4j.SessionConfig{})
	defer session.Close(c.ctx)

	_, err := session.ExecuteWrite(c.ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(c.ctx, "MATCH (n) DETACH DELETE n", nil)
		return nil, err
	})

	return err
}

// ExecuteQuery runs a Cypher query
func (c *Neo4jClient) ExecuteQuery(query string, params map[string]any) (neo4j.ResultWithContext, error) {
	session := c.driver.NewSession(c.ctx, neo4j.SessionConfig{})
	defer session.Close(c.ctx)

	result, err := session.Run(c.ctx, query, params)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ExecuteWriteQuery runs a write query in a transaction
func (c *Neo4jClient) ExecuteWriteQuery(query string, params map[string]any) error {
	session := c.driver.NewSession(c.ctx, neo4j.SessionConfig{})
	defer session.Close(c.ctx)

	_, err := session.ExecuteWrite(c.ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(c.ctx, query, params)
		return nil, err
	})

	return err
}

// ExecuteReadQuery runs a read query and returns results
func (c *Neo4jClient) ExecuteReadQuery(query string, params map[string]any) ([]*neo4j.Record, error) {
	session := c.driver.NewSession(c.ctx, neo4j.SessionConfig{})
	defer session.Close(c.ctx)

	result, err := session.ExecuteRead(c.ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(c.ctx, query, params)
		if err != nil {
			return nil, err
		}

		records, err := result.Collect(c.ctx)
		if err != nil {
			return nil, err
		}

		return records, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]*neo4j.Record), nil
}
