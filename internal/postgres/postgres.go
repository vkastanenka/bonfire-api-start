package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool parses a connection URL, configures connection pool,
// initializes the pool, and verifies connectivity with a ping.
func NewPool(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
	// Validate string
	if connStr == "" {
		return nil, fmt.Errorf("postgres connection string cannot be empty")
	}

	// Track time
	start := time.Now()
	slog.Info("initializing postgres connection pool")

	// Parse connection string
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	// Pool tuning parameters
	config.MaxConns = 25
	config.MinConns = 2
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	// Define init ctx
	initCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Setup new pool
	dbPool, err := pgxpool.NewWithConfig(initCtx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	// Ping to ensure connectivity
	if err := dbPool.Ping(initCtx); err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("postgres connection verification failed: %w", err)
	}

	// Return pool
	slog.Info("postgres connection established", slog.Duration("duration", time.Since(start)))
	return dbPool, nil
}
