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
	"github.com/joho/godotenv"
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
	// Load env
	if err := godotenv.Load(); err != nil {
		slog.Warn("no .env file found, falling back to system environment variables")
	}

	// Load config
	cfg := config.Load()

	// Define ctx
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Load db
	dbPool, err := database.NewPostgresPool(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer dbPool.Close()

	// Verify db connection
	if err := dbPool.Ping(ctx); err != nil {
		slog.Error("database connection verification failed", "error", err)
		return err
	}

	// Load cache
	rdb, err := cache.NewRedisClient(cfg.RedisURL)
	if err != nil {
		return err
	}
	defer rdb.Close()

	// Verify cache connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("redis connection verification failed", "error", err)
		return err
	}

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

	// Setup application
	app := &Application{
		Config:      cfg,
		DB:          dbPool,
		Redis:       rdb,
		RateLimiter: rateLimiter,
		AuthHandler: authHandler,
	}

	// Serve app safely
	return app.Serve(ctx)
}
