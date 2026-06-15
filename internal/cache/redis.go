package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient initializes and pings a new Redis client instance.
func NewRedisClient(addr string) (*redis.Client, error) {
	// Provide a sensible default if the address wasn't passed in
	if addr == "" {
		addr = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	// Use a focused timeout context for the initial ping check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ping to ensure connectivity
	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close() // Explicitly close the connection pool if ping fails
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return rdb, nil
}
