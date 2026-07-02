package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool parses the connection string, configures connection pool dynamics,
// initializes the pool, and verifies connectivity with a ping.
func NewPool(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
	if connStr == "" {
		return nil, fmt.Errorf("postgres connection string cannot be empty")
	}

	start := time.Now()
	slog.Info("initializing postgres connection pool")

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	// Pool tuning parameters
	config.MaxConns = 25
	config.MinConns = 2
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	initCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	dbPool, err := pgxpool.NewWithConfig(initCtx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	if err := dbPool.Ping(initCtx); err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("postgres connection verification failed: %w", err)
	}

	slog.Info("postgres connection established", slog.Duration("duration", time.Since(start)))
	return dbPool, nil
}
