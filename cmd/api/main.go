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
	"bonfire-api/internal/bootstrap"
	"bonfire-api/internal/config"
	"bonfire-api/internal/email"
	"bonfire-api/internal/health"
	"bonfire-api/internal/logger"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/validator"
	"bonfire-api/internal/worker"

	"github.com/go-redis/redis_rate/v10"
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

	// Execute in run() to respect defers
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
	cfg, err := config.Load()

	// Define ctx
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	// Init db
	pdbPool, err := bootstrap.InitPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pdbPool.Close()

	// Init redis
	rdb, err := bootstrap.InitRedis(ctx, cfg.RedisURL)
	if err != nil {
		return err
	}
	defer rdb.Close()

	// Setup data layer
	store := repository.NewStore(pdbPool)
	queries := repository.New(pdbPool)

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
	healthHandler := health.NewHandler(pdbPool, rdb)

	// Setup application container
	app := &Application{
		Config:        cfg,
		DB:            pdbPool,
		Redis:         rdb,
		RateLimiter:   rateLimiter,
		AuthHandler:   authHandler,
		HealthHandler: healthHandler,
	}

	// Serve application safely
	return app.Serve(ctx)
}
