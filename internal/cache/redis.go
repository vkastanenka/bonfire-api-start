package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient parses a connection URL, applies connection pool limits,
// and verifies connection by pinging the instance.
func NewRedisClient(ctx context.Context, redisURL string) (*redis.Client, error) {
	// Validate redisURL
	if redisURL == "" {
		return nil, fmt.Errorf("redis connection string cannot be empty")
	}

	// Parse URL
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis url: %w", err)
	}

	// Init pool settings
	opt.PoolSize = 20
	opt.MinIdleConns = 2
	opt.ConnMaxIdleTime = 30 * time.Minute
	opt.ConnMaxLifetime = 1 * time.Hour

	// Init timeout from ctx
	initCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Init client
	rdb := redis.NewClient(opt)

	// Verify connection with a ping
	if err := rdb.Ping(initCtx).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("redis connection verification failed: %w", err)
	}

	return rdb, nil
}
