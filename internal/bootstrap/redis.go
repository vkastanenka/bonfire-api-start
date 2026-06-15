package bootstrap

import (
	"bonfire-api/internal/cache"
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// InitRedis handles Redis initialization and verification
func InitRedis(ctx context.Context, url string) (*redis.Client, error) {
	// Start
	start := time.Now()
	slog.Info("initializing redis client")

	// Init new client
	rdb, err := cache.NewRedisClient(url)
	if err != nil {
		slog.Error("redis initialization failed", "error", err)
		return nil, err
	}

	// Ping to verify connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("redis connection verification failed", "error", err)
		rdb.Close()
		return nil, err
	}

	// Finish
	slog.Info("redis connection established", "duration", time.Since(start))
	return rdb, nil
}
