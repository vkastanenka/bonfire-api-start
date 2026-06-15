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

// main is the entry point for the API. It utilizes the "hollow main" pattern,
// delegating execution to run() to ensure that deferred cleanup functions
// are honored prior to the application exiting via os.Exit.
func main() {
	// Configure global slog instance
	logger.InitLogger()

	// Hollow main: Execution is delegated to run() so defers are respected.
	if err := run(); err != nil {
		slog.Error("startup failed", "error", err.Error())
		os.Exit(1)
	}
}

// run orchestrates the bootstrapping sequence of the application. It loads
// configurations, initializes infrastructure connections, wires dependencies,
// starts background workers, and launches the HTTP server with graceful
// shutdown capabilities.
func run() error {
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found, falling back to system environment variables")
	}

	cfg := config.Load()

	// 2. Establish Infrastructure Connection Pools
	dbPool, err := database.NewPostgresPool(cfg.DatabaseURL)
	if err != nil {
		return err // Returns error instead of log.Fatal, executing defers!
	}
	defer dbPool.Close()

	rdb, err := cache.NewRedisClient(cfg.RedisURL)
	if err != nil {
		return err
	}
	defer rdb.Close()

	// 3. Setup Architecture Data Layers
	store := repository.NewStore(dbPool)
	queries := repository.New(dbPool)
	rateLimiter := redis_rate.NewLimiter(rdb)

	// 4. Resolve Domain Configuration Objects
	tokenConfig := auth.TokenConfig{
		AccessSecret:        cfg.AccessSecret,
		RefreshSecret:       cfg.RefreshSecret,
		VerificationSecret:  cfg.VerificationSecret,
		PasswordResetSecret: cfg.PasswordResetSecret,
	}

	// 5. Initialize Mail Engine and Domain Services
	mailer := email.NewMailer(cfg)
	authService := auth.NewAuthService(store, tokenConfig)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 6. Instantiate Background Outbox System Threads
	outboxWorker := worker.NewOutboxWorker(queries, mailer, 1*time.Second, 10)
	outboxWorker.Start(ctx)
	defer outboxWorker.Stop() // This will now cleanly finish its batch on exit!

	// 7. Assemble Handler Layer Dependencies
	val := validator.New()
	authHandler := auth.NewAuthHandler(authService, val)

	// 8. Bind the Application Control Container
	app := &Application{
		Config:      cfg,
		DB:          dbPool,
		Redis:       rdb,
		RateLimiter: rateLimiter,
		AuthHandler: authHandler,
	}

	// 9. Start the server safely
	return app.Serve(ctx)
}
