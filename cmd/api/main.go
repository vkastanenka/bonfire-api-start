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
	"bonfire-api/internal/repository"
	"bonfire-api/internal/validator"
	"bonfire-api/internal/worker"

	"github.com/go-redis/redis_rate/v10"
	"github.com/joho/godotenv"
)

func main() {
	initLogger()

	// 1. Hollow main: Execution is delegated to run() so defers are respected.
	if err := run(); err != nil {
		slog.Error("startup failed", "error", err.Error())
		os.Exit(1)
	}
}

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

func initLogger() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}
