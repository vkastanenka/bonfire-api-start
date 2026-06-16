package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresPool parses the connection string, sets up pool configurations,
// and verifies connection by pinging the database.
func NewPostgresPool(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
	// Validate connStr
	if connStr == "" {
		return nil, fmt.Errorf("postgres connection string cannot be empty")
	}

	// Parse connStr
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	// Init pool settings
	config.MaxConns = 25
	config.MinConns = 2
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	// Init timeout from ctx
	initCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Init pool
	dbPool, err := pgxpool.NewWithConfig(initCtx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	// Verify connection with a ping
	if err := dbPool.Ping(initCtx); err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("postgres connection verification failed: %w", err)
	}

	return dbPool, nil
}
