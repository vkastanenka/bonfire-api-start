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

// @title           Bonfire API
// @version         1.0
// @description     The full-stack, real-time chat application backend API.

// @contact.name   Victoria Kastanenka
// @contact.email  vkastanenka@gmail.com

// @host      localhost:8080
// @BasePath  /api/v1
func main() {
	// Setup config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	// Setup global slog instance
	logger.Init(logger.Config{
		Level:     slog.LevelInfo,
		AddSource: cfg.IsDevelopment(),
	})

	// Execute run()
	if err := run(cfg); err != nil {
		slog.Error("startup failed", slog.Any("error", err))
		os.Exit(1)
	}
}

// run
func run(cfg *config.Config) error {
	// Setup ctx
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer stop()

	// Setup postgres
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

	// Setup redis
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

	// Setup data layer
	store := repository.NewStore(pdbPool)

	// Setup app services
	rateLimiter := redis_rate.NewLimiter(rdbClient)
	redisManager := redis.NewManager(rdbClient)
	tokenService := token.NewService(
		cfg.AccessSecret,
		cfg.RefreshSecret,
		cfg.VerificationSecret,
		cfg.PasswordResetSecret,
		cfg.PasswordMFASecret,
	)

	// Setup domain services
	sessionService := session.NewService(store)
	userService := user.NewService(store)
	authService := auth.NewService(
		store,
		redisManager.(redis.Store),
		sessionService,
		tokenService,
		userService,
	)

	// Setup presentation layer
	authHandler := auth.NewHandler(authService)
	userHandler := user.NewHandler(userService)

	// Setup application container
	app := &Application{
		Config:      cfg,
		DB:          pdbPool,
		Redis:       rdbClient,
		RateLimiter: rateLimiter,
		Handlers: struct {
			Auth  *auth.Handler
			Users *user.Handler
		}{
			Auth:  authHandler,
			Users: userHandler,
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

// // run
// func run(cfg *config.Config) error {
// 	// Setup ctx
// 	ctx, stop := signal.NotifyContext(
// 		context.Background(),
// 		syscall.SIGINT,
// 		syscall.SIGTERM,
// 		syscall.SIGQUIT,
// 	)
// 	defer stop()

// 	// Setup postgres
// 	pdbPool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
// 	if err != nil {
// 		return err
// 	}
// 	defer pdbPool.Close()

// 	// Setup redis
// 	rdb, err := redis.NewClient(ctx, cfg.RedisURL)
// 	if err != nil {
// 		return err
// 	}
// 	defer rdb.Close()

// 	// Setup data layer
// 	store := repository.NewStore(pdbPool)

// 	// Setup helper services
// 	val := validator.New()
// 	redisManager := redis.NewManager(rdb)
// 	rateLimiter := redis_rate.NewLimiter(rdb)
// 	tokenService := token.NewService(cfg.AccessSecret, cfg.RefreshSecret, cfg.VerificationSecret, cfg.PasswordResetSecret, cfg.PasswordMFASecret)

// 	// Setup domain services
// 	messageService := message.NewService(store)
// 	outboxEventsService := outbox.NewService(store)
// 	sessionService := session.NewService(store)
// 	userService := user.NewService(store)
// 	authService := auth.NewService(
// 		store,
// 		redisManager.(redis.Store),
// 		sessionService,
// 		tokenService,
// 		userService,
// 	)

// 	// Setup real-time gateway core
// 	gatewayHub := gateway.NewHub(redisManager, store, messageService)
// 	go gatewayHub.Run(ctx) // Spawns background pubsub engine loop natively
// 	gatewayHandler := gateway.NewHandler(gatewayHub)

// 	// Setup background workers
// 	mailer := email.NewMailer(email.Config{
// 		ResendAPIKey: cfg.ResendApiKey,
// 		FromAddress:  cfg.EmailFromAddress,
// 		FrontendURL:  cfg.FrontendURL,
// 		OverrideTo:   cfg.EmailOverrideTo,
// 	})
// 	outboxWorker := worker.NewOutboxWorker(store.Queries, mailer, 5*time.Second, 10)
// 	outboxWorker.Start(ctx)
// 	defer outboxWorker.Stop()

// 	// Setup presentation layer
// 	authHandler := auth.NewHandler(authService, val)
// 	healthHandler := health.NewHandler(pdbPool, rdb)
// 	outboxEventsHandler := outbox.NewHandler(outboxEventsService)
// 	userHandler := user.NewHandler(userService, val)

// 	// Setup application container
// 	app := &Application{
// 		Config:      cfg,
// 		DB:          pdbPool,
// 		Redis:       rdb,
// 		RateLimiter: rateLimiter,
// 		Handlers: struct {
// 			Auth         *auth.Handler
// 			Health       *health.Handler
// 			OutboxEvents *outbox.Handler
// 			Users        *user.Handler
// 			Gateway      *gateway.Handler
// 		}{
// 			Auth:         authHandler,
// 			Health:       healthHandler,
// 			OutboxEvents: outboxEventsHandler,
// 			Users:        userHandler,
// 			Gateway:      gatewayHandler,
// 		},
// 		Services: struct {
// 			Token *token.Service
// 		}{
// 			Token: tokenService,
// 		},
// 	}

// 	// Serve application safely
// 	return app.Serve(ctx)
// }
