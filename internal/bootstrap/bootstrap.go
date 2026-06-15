package bootstrap

import (
	"context"
	"log/slog"

	"bonfire-api/internal/cache"
	"bonfire-api/internal/config"
	"bonfire-api/internal/database"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

// InitConfig handles environment and config loading
func InitConfig() (*config.Config, error) {
	godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration load failed", "error", err)
		return nil, err
	}
	return cfg, nil
}

// InitDatabase handles DB initialization and verification
func InitDatabase(ctx context.Context, url string) (*pgxpool.Pool, error) {
	slog.Info("initializing database connection pool")
	dbPool, err := database.NewPostgresPool(url)
	if err != nil {
		slog.Error("database initialization failed", "error", err)
		return nil, err
	}

	if err := dbPool.Ping(ctx); err != nil {
		slog.Error("database connection verification failed", "error", err)
		dbPool.Close()
		return nil, err
	}

	slog.Info("database connection established")
	return dbPool, nil
}

// InitRedis handles Redis initialization and verification
func InitRedis(ctx context.Context, url string) (*redis.Client, error) {
	slog.Info("initializing redis client")
	rdb, err := cache.NewRedisClient(url)
	if err != nil {
		slog.Error("redis initialization failed", "error", err)
		return nil, err
	}

	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("redis connection verification failed", "error", err)
		rdb.Close()
		return nil, err
	}

	slog.Info("redis connection established")
	return rdb, nil
}
