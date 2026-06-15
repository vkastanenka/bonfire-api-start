package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresPool initializes a connection pool to the PostgreSQL database.
// It accepts a connection string, handles the connection setup, and pings the DB to confirm it's ready.
func NewPostgresPool(connStr string) (*pgxpool.Pool, error) {
	if connStr == "" {
		return nil, fmt.Errorf("database connection string cannot be empty")
	}

	// Create a localized context with a timeout for the initialization phase
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Parse configuration and instantiate the pool connection
	dbPool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to create database pool: %w", err)
	}

	// Ping the connection to ensure it's structurally alive and valid
	if err := dbPool.Ping(ctx); err != nil {
		dbPool.Close() // Clean up the pool if the ping fails
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	return dbPool, nil
}
