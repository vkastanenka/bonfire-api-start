/*
Package main provides the entry point for the Bonfire API.

This file acts as the centralized dependency injection container and
application orchestrator. It executes a strict, bottom-up bootstrapping
sequence:

 1. Configuring global telemetry and logging.
 2. Loading environment configurations.
 3. Establishing resilient infrastructure connections (Postgres, Redis).
 4. Wiring data layers, domain services, and HTTP handlers.
 5. Spooling up background system workers.
 6. Initializing OS-signal listeners for graceful degradation.

By utilizing a "hollow main" pattern, this file guarantees that all
resources, background threads, and connection pools are safely closed
via deferred functions prior to the application exiting.
*/
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bonfire-api/internal/auth"
	"bonfire-api/internal/cache"
	"bonfire-api/internal/config"
	"bonfire-api/internal/database"
	"bonfire-api/internal/email"
	"bonfire-api/internal/logger"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/validator"
	"bonfire-api/internal/worker"

	"github.com/go-redis/redis_rate/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

// @title           Bonfire API
// @version         1.0
// @description     The full-stack, real-time chat application backend API.

// @contact.name   Victoria Kastanenka
// @contact.email  vkastanenka@gmail.com

// @host      localhost:8080
// @BasePath  /api/v1
func main() {
	// Configure global slog instance
	logger.InitLogger()

	// Hollow main: Execution is delegated to run() so defers are respected.
	if err := run(); err != nil {
		slog.Error("startup failed", "error", err)
		os.Exit(1)
	}
}

// run orchestrates the bootstrapping sequence of the application. It loads
// configurations, initializes infrastructure connections, wires dependencies,
// starts background workers, and launches the HTTP server with graceful
// shutdown capabilities.
func run() error {
	// Init config
	cfg, err := initConfig()
	if err != nil {
		return err
	}

	// Define ctx
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Init db
	dbPool, err := initDatabase(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer dbPool.Close()

	// Init redis
	rdb, err := initRedis(ctx, cfg.RedisURL)
	if err != nil {
		return err
	}
	defer rdb.Close()

	// Setup data layer
	store := repository.NewStore(dbPool)
	queries := repository.New(dbPool)

	// Setup middleware services
	rateLimiter := redis_rate.NewLimiter(rdb)
	val := validator.New()

	// Setup domain services
	mailer := email.NewMailer(cfg)
	authService := auth.NewAuthService(store, auth.TokenConfig{
		AccessSecret:        cfg.AccessSecret,
		RefreshSecret:       cfg.RefreshSecret,
		VerificationSecret:  cfg.VerificationSecret,
		PasswordResetSecret: cfg.PasswordResetSecret,
	})

	// Setup background workers
	outboxWorker := worker.NewOutboxWorker(queries, mailer, 1*time.Second, 10)
	outboxWorker.Start(ctx)
	defer outboxWorker.Stop()

	// Setup presentation layer
	authHandler := auth.NewAuthHandler(authService, val)

	// Setup application container
	app := &Application{
		Config:      cfg,
		DB:          dbPool,
		Redis:       rdb,
		RateLimiter: rateLimiter,
		AuthHandler: authHandler,
	}

	// Serve appplication safely
	return app.Serve(ctx)
}

// -----------------------------------------------------------------------------
// Private Bootstrapping Helpers
// -----------------------------------------------------------------------------

// initConfig loads environment variables and parses them into the Config struct.
func initConfig() (*config.Config, error) {
	godotenv.Load() // Silent fallback allowed; validator acts as gatekeeper

	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration load failed", "error", err)
		return nil, err
	}

	return cfg, nil
}

// initDatabase initializes the connection pool and verifies network connectivity.
// Note: Adjust the *pgxpool.Pool return type if your database.NewPostgresPool
// returns a custom struct wrapper.
func initDatabase(ctx context.Context, url string) (*pgxpool.Pool, error) {
	slog.Info("initializing database connection pool")
	dbPool, err := database.NewPostgresPool(url)
	if err != nil {
		slog.Error("database initialization failed", "error", err)
		return nil, err
	}

	if err := dbPool.Ping(ctx); err != nil {
		slog.Error("database connection verification failed", "error", err)
		dbPool.Close() // Clean up since run() won't get the defer
		return nil, err
	}

	slog.Info("database connection established")
	return dbPool, nil
}

// initRedis initializes the caching client and verifies network connectivity.
// Note: Adjust the *redis.Client return type if your cache.NewRedisClient
// returns a custom struct wrapper.
func initRedis(ctx context.Context, url string) (*redis.Client, error) {
	slog.Info("initializing redis client")
	rdb, err := cache.NewRedisClient(url)
	if err != nil {
		slog.Error("redis initialization failed", "error", err)
		return nil, err
	}

	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("redis connection verification failed", "error", err)
		rdb.Close() // Clean up since run() won't get the defer
		return nil, err
	}

	slog.Info("redis connection established")
	return rdb, nil
}
