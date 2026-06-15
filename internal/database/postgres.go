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
		return nil, fmt.Errorf("postgres connection string cannot be empty")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Init db pool
	dbPool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("postgres pool init failed: %w", err)
	}

	// Ping to verify connection
	if err := dbPool.Ping(ctx); err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("postgres ping failed: %w", err)
	}

	return dbPool, nil
}
