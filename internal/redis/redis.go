package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewClient parses a connection URL, configures connection pooling constraints,
// and verifies instance readiness via a diagnostic ping.
func NewClient(ctx context.Context, redisURL string) (*redis.Client, error) {
	if redisURL == "" {
		return nil, fmt.Errorf("redis connection string cannot be empty")
	}

	start := time.Now()
	slog.Info("initializing redis client")

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis url: %w", err)
	}

	// Client tuning parameters
	opt.PoolSize = 20
	opt.MinIdleConns = 2
	opt.ConnMaxIdleTime = 30 * time.Minute
	opt.ConnMaxLifetime = 1 * time.Hour

	initCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rdb := redis.NewClient(opt)

	// Cleaned up double-ping bug by keeping verification fully contained here
	if err := rdb.Ping(initCtx).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("redis connection verification failed: %w", err)
	}

	slog.Info("redis connection established", slog.Duration("duration", time.Since(start)))
	return rdb, nil
}
