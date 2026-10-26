package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"bonfire-api/internal/auth"
	"bonfire-api/internal/bootstrap"
	"bonfire-api/internal/config"
	"bonfire-api/internal/db"
	"bonfire-api/internal/email"
	"bonfire-api/internal/gateway"
	"bonfire-api/internal/health"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/pkg/logger"
	"bonfire-api/internal/pkg/validator"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/shield"
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

	errs.SetDomain(cfg.Domain)

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

	dbClient, err := db.NewConn(ctx, db.ConnConfig{
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
	defer dbClient.Close()

	cacheClient, err := redis.NewConn(ctx, redis.ConnConfig{
		ConnString:      cfg.RedisURL,
		PoolSize:        cfg.RedisPoolSize,
		MinIdleConns:    cfg.RedisMinIdleConns,
		ConnMaxIdleTime: cfg.RedisConnMaxIdleTime,
		ConnMaxLifetime: cfg.RedisConnMaxLifetime,
	})
	if err != nil {
		return err
	}
	defer cacheClient.Close()

	tokenProvider, err := token.NewProvider(token.ProviderConfig{
		Issuer: cfg.AppName,
		Access: token.TypeConfig{
			Secret: cfg.JWTAccessSecret,
			TTL:    cfg.JWTAccessTTL,
		},
		Refresh: token.TypeConfig{
			Secret: cfg.JWTRefreshSecret,
			TTL:    cfg.JWTRefreshTTL,
		},
		EmailVerify: token.TypeConfig{
			Secret: cfg.JWTEmailVerifySecret,
			TTL:    cfg.JWTEmailVerifyTTL,
		},
		PasswordReset: token.TypeConfig{
			Secret: cfg.JWTPasswordResetSecret,
			TTL:    cfg.JWTPasswordResetTTL,
		},
	})
	if err != nil {
		return err
	}
	dbStore := db.NewStore(dbClient)
	rateLimiter := redis_rate.NewLimiter(cacheClient)

	val := validator.New()
	val.RegisterAliases(user.ValidationAliases())
	bind := httpio.NewBind(val)

	mailer := email.NewMailer(email.Config{
		ResendAPIKey: cfg.ResendApiKey,
		FromAddress:  cfg.EmailFromAddress,
		FrontendURL:  cfg.FrontendURL,
		OverrideTo:   cfg.EmailOverrideTo,
	})

	authCache := auth.NewCache(cacheClient)
	sessionCache := auth.NewSessionCache(cacheClient)
	shieldCache := shield.NewCache(cacheClient)
	tokenCache := token.NewCache(cacheClient)
	userCache := user.NewCache(cacheClient)

	outboxRepo := outbox.NewRepository(dbStore)
	sessionRepo := auth.NewSessionRepository(dbStore)
	userRepo := user.NewRepository(dbStore)

	cachedUserRepo := user.NewCachedRepository(userCache, userRepo)

	gatewayCache := gateway.NewCache(cacheClient)
	pubsub := redis.NewPubSub(cacheClient)
	broadcaster := gateway.NewBroadcaster(gatewayCache, pubsub)

	authSvc := auth.NewService(
		authCache,
		sessionCache,
		sessionRepo,
		broadcaster,
		outboxRepo,
		shieldCache,
		tokenCache,
		tokenProvider,
		userCache,
		userRepo,
		cachedUserRepo,
		dbStore,
	)
	bootstrapSvc := bootstrap.NewService(
		cachedUserRepo,
	)
	gatewaySvc := gateway.NewService(
		gatewayCache,
		broadcaster,
		pubsub,
	)
	userSvc := user.NewService(
		userCache,
		userRepo,
		cachedUserRepo,
		authCache,
		broadcaster,
		outboxRepo,
		sessionCache,
		sessionRepo,
		dbStore,
	)

	authHandler := auth.NewHandler(authSvc, bind)
	bootstrapHandler := bootstrap.NewHandler(bootstrapSvc, bind)
	healthHandler := health.NewHandler(cacheClient, dbClient)
	userHandler := user.NewHandler(userSvc, bind)

	wsHub := gateway.NewHub(gatewaySvc)
	go wsHub.Run(ctx)

	gatewayHandler := gateway.NewHandler(wsHub, gatewaySvc, bind)

	outboxWorker, err := outbox.NewWorker(
		outboxRepo,
		cfg.OutboxPollInterval,
		cfg.OutboxLeaseDuration,
		cfg.OutboxBatchSize,
		cfg.OutboxMaxWorkers,
	)
	if err != nil {
		return err
	}

	auth.RegisterEvents(outboxWorker, mailer, broadcaster)

	outboxWorker.Start(ctx)
	defer outboxWorker.Stop(ctx)

	app := &Application{
		Config:      cfg,
		RateLimiter: rateLimiter,
		Tokens:      tokenProvider,
		Handlers: Handlers{
			Auth:      authHandler,
			Bootstrap: bootstrapHandler,
			Gateway:   gatewayHandler,
			Health:    healthHandler,
			User:      userHandler,
		},
	}

	return app.Serve(ctx)
}
