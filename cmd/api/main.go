package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"bonfire-api/internal/auth"
	"bonfire-api/internal/config"
	"bonfire-api/internal/logger"
	"bonfire-api/internal/postgres"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/session"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"

	"github.com/go-redis/redis_rate/v10"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Init(logger.Config{
		Level:     slog.LevelInfo,
		AddSource: cfg.IsDevelopment(),
	})

	if err := run(cfg); err != nil {
		slog.Error("startup failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(cfg *config.Config) error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer stop()

	pdbPool, err := postgres.NewPool(ctx, postgres.Config{
		ConnString:      cfg.DatabaseURL,
		MaxConns:        cfg.DBMaxConns,
		MinConns:        cfg.DBMinConns,
		MaxConnLifetime: cfg.DBMaxConnLifetime,
		MaxConnIdleTime: cfg.DBMaxConnIdleTime,
		HealthCheck:     cfg.DBHealthCheck,
	})
	if err != nil {
		return err
	}
	defer pdbPool.Close()

	rdbClient, err := redis.NewClient(ctx, redis.Config{
		ConnString:      cfg.RedisURL,
		PoolSize:        cfg.RedisPoolSize,
		MinIdleConns:    cfg.RedisMinIdleConns,
		ConnMaxIdleTime: cfg.RedisConnMaxIdleTime,
		ConnMaxLifetime: cfg.RedisConnMaxLifetime,
	})
	if err != nil {
		return err
	}
	defer rdbClient.Close()

	store := repository.NewStore(pdbPool)
	rateLimiter := redis_rate.NewLimiter(rdbClient)
	tokenManager, err := token.NewManager(token.Config{
		AccessSecret:  cfg.AccessSecret,
		RefreshSecret: cfg.RefreshSecret,
		Issuer:        cfg.TokenIssuer,
	})
	if err != nil {
		return err
	}

	sessionService := session.NewService(store)
	userService := user.NewService(store)
	authService := auth.NewService(
		store,
		sessionService,
		tokenManager,
		userService,
	)

	authHandler := auth.NewHandler(authService)

	app := &Application{
		Config:      cfg,
		DB:          pdbPool,
		Redis:       rdbClient,
		RateLimiter: rateLimiter,
		Handlers: Handlers{
			Auth: authHandler,
		},
	}

	return app.Serve(ctx)
}
