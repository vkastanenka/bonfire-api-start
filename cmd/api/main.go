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
	"bonfire-api/internal/cache"
	"bonfire-api/internal/config"
	"bonfire-api/internal/email"
	"bonfire-api/internal/health"
	"bonfire-api/internal/logger"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/session"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"
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

	// Setup helper services
	val := validator.New()
	cacheManager := cache.NewManager(rdb)
	rateLimiter := redis_rate.NewLimiter(rdb)
	tokenService := token.NewService(cfg.AccessSecret, cfg.RefreshSecret, cfg.VerificationSecret, cfg.PasswordResetSecret, cfg.PasswordMFASecret)

	// Setup domain services
	outboxEventsService := outbox.NewService(store)
	sessionService := session.NewService(store)
	userService := user.NewService(store)
	authService := auth.NewService(
		store,
		cacheManager.(cache.Store),
		sessionService,
		tokenService,
		userService,
	)
	// chatService := chat.NewService(chatRepo, appCache.(cache.MessageBus), appCache.(cache.PresenceTracker))

	// Setup background workers
	mailer := email.NewMailer(email.Config{
		ResendAPIKey: cfg.ResendApiKey,
		FromAddress:  cfg.EmailFromAddress,
		FrontendURL:  cfg.FrontendURL,
		OverrideTo:   cfg.EmailOverrideTo,
	})
	outboxWorker := worker.NewOutboxWorker(queries, mailer, 5*time.Second, 10)
	outboxWorker.Start(ctx)
	defer outboxWorker.Stop()

	// Setup presentation layer
	authHandler := auth.NewHandler(authService, val)
	healthHandler := health.NewHandler(pdbPool, rdb)
	outboxEventsHandler := outbox.NewHandler(outboxEventsService)
	userHandler := user.NewHandler(userService, val)

	// Setup application container
	app := &Application{
		Config:      cfg,
		DB:          pdbPool,
		Redis:       rdb,
		RateLimiter: rateLimiter,
		Handlers: struct {
			Auth         *auth.Handler
			Health       *health.Handler
			OutboxEvents *outbox.Handler
			Users        *user.Handler
		}{
			Auth:         authHandler,
			Health:       healthHandler,
			OutboxEvents: outboxEventsHandler,
			Users:        userHandler,
		},
		Services: struct {
			Token *token.Service
		}{
			Token: tokenService,
		},
	}

	// Serve application safely
	return app.Serve(ctx)
}
