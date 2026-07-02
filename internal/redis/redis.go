package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewClient parses a connection URL, configures connection pool,
// initializes the pool, and verifies connectivity with a ping.
func NewClient(ctx context.Context, redisURL string) (*redis.Client, error) {
	// Validate string
	if redisURL == "" {
		return nil, fmt.Errorf("redis connection string cannot be empty")
	}

	// Track time
	start := time.Now()
	slog.Info("initializing redis client")

	// Parse connection string
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis url: %w", err)
	}

	// Client tuning parameters
	opt.PoolSize = 20
	opt.MinIdleConns = 2
	opt.ConnMaxIdleTime = 30 * time.Minute
	opt.ConnMaxLifetime = 1 * time.Hour

	// Define init ctx
	initCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Setup new client
	rdb := redis.NewClient(opt)

	// Ping to ensure connectivity
	if err := rdb.Ping(initCtx).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("redis connection verification failed: %w", err)
	}

	// Return client
	slog.Info("redis connection established", slog.Duration("duration", time.Since(start)))
	return rdb, nil
}
