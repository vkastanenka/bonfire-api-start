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
	"bonfire-api/internal/health"
	"bonfire-api/internal/logger"
	"bonfire-api/internal/outbox_events"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"
	"bonfire-api/internal/user_profile"
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

// run
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
	tokenManager := token.NewJWTManager()

	// Setup domain services
	authService := auth.NewAuthService(store, tokenManager, auth.TokenConfig{
		AccessSecret:        cfg.AccessSecret,
		RefreshSecret:       cfg.RefreshSecret,
		VerificationSecret:  cfg.VerificationSecret,
		PasswordResetSecret: cfg.PasswordResetSecret,
	})
	outboxEventsService := outbox_events.NewService(store)
	userService := user.NewUserService(store)
	userProfileService := user_profile.NewService(store)

	// Setup background workers
	outboxWorker := worker.NewOutboxWorker(queries, 5*time.Second, 10)
	outboxWorker.Start(ctx)
	defer outboxWorker.Stop()

	// Setup presentation layer
	authHandler := auth.NewHandler(authService, val)
	healthHandler := health.NewHandler(pdbPool, rdb)
	outboxEventsHandler := outbox_events.NewHandler(outboxEventsService, val)
	userHandler := user.NewHandler(userService, val)
	userProfileHandler := user_profile.NewHandler(userProfileService)

	// Setup application container
	app := &Application{
		Config:       cfg,
		DB:           pdbPool,
		Redis:        rdb,
		RateLimiter:  rateLimiter,
		TokenManager: tokenManager,
		Handlers: struct {
			Auth         *auth.AuthHandler
			Health       *health.Handler
			OutboxEvents *outbox_events.Handler
			User         *user.Handler
			UserProfile  *user_profile.Handler
		}{
			Auth:         authHandler,
			Health:       healthHandler,
			OutboxEvents: outboxEventsHandler,
			User:         userHandler,
			UserProfile:  userProfileHandler,
		},
	}

	// Serve application safely
	return app.Serve(ctx)
}
