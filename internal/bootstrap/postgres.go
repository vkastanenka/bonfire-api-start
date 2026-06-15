package bootstrap

import (
	"bonfire-api/internal/database"
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// InitPostgres handles DB initialization and verification
func InitPostgres(ctx context.Context, url string) (*pgxpool.Pool, error) {
	// Start
	start := time.Now()
	slog.Info("initializing postgres connection pool")

	// Init new client
	dbPool, err := database.NewPostgresPool(url)
	if err != nil {
		slog.Error("postgres init failed", "error", err)
		return nil, err
	}

	// Ping to verify connection
	if err := dbPool.Ping(ctx); err != nil {
		slog.Error("postgres connection verification failed", "error", err)
		dbPool.Close()
		return nil, err
	}

	// Finish
	slog.Info("postgres connection established", "duration", time.Since(start))
	return dbPool, nil
}
