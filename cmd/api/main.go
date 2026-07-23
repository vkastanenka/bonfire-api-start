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
	"bonfire-api/internal/db"
	"bonfire-api/internal/email"
	"bonfire-api/internal/handler"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/logger"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/store"
	"bonfire-api/internal/token"
	"bonfire-api/internal/validator"

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

	dbConn, err := db.NewConn(ctx, db.ConnConfig{
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
	defer dbConn.Close()

	cacheConn, err := cache.NewConn(ctx, cache.ConnConfig{
		ConnString:      cfg.RedisURL,
		PoolSize:        cfg.RedisPoolSize,
		MinIdleConns:    cfg.RedisMinIdleConns,
		ConnMaxIdleTime: cfg.RedisConnMaxIdleTime,
		ConnMaxLifetime: cfg.RedisConnMaxLifetime,
	})
	if err != nil {
		return err
	}
	defer cacheConn.Close()

	tokens, err := token.NewProvider(token.Config{
		Issuer: cfg.TokenIssuer,
		Access: token.VariantConfig{
			Secret: cfg.AccessSecret,
			TTL:    cfg.JWTAccessTTL,
		},
		Refresh: token.VariantConfig{
			Secret: cfg.RefreshSecret,
			TTL:    cfg.JWTRefreshTTL,
		},
		EmailVerify: token.VariantConfig{
			Secret: cfg.EmailVerifySecret,
			TTL:    cfg.JWTEmailVerifyTTL,
		},
		PasswordReset: token.VariantConfig{
			Secret: cfg.PasswordResetSecret,
			TTL:    cfg.JWTPasswordResetTTL,
		},
	})
	if err != nil {
		return err
	}
	dbStore := db.NewStore(dbConn)
	cacheStore := cache.NewStore(cacheConn)
	rateLimiter := redis_rate.NewLimiter(cacheConn)
	val := validator.New()
	bind := httpio.NewBind(val)
	mailer := email.NewMailer(email.Config{
		ResendAPIKey: cfg.ResendApiKey,
		FromAddress:  cfg.EmailFromAddress,
		FrontendURL:  cfg.FrontendURL,
		OverrideTo:   cfg.EmailOverrideTo,
	})

	outboxRepo := repository.NewOutbox(dbStore)
	// relationshipRepo := repository.NewRelationship(dbStore)
	sessionRepo := repository.NewSession(dbStore)
	userRepo := repository.NewUser(dbStore)

	// presenceStore := store.NewPresence(cacheStore, cfg.PresenceTTL)
	sessionStore := store.NewSession(cacheStore)
	shieldStore := store.NewShield(cacheStore)
	ticketStore := store.NewTicket(cacheStore)

	// relationshipSvc := relationship.NewService(relationshipRepo)
	// presenceSvc := presence.NewService(presenceStore)
	// userSvc := user.NewService(userRepo)
	authSvc := auth.NewService(
		outboxRepo,
		sessionRepo,
		userRepo,
		sessionStore,
		shieldStore,
		ticketStore,
		tokens,
		dbStore,
	)

	outboxWorker := outbox.NewWorker(
		outboxRepo,
		2*time.Second,
		int32(10),
		int32(10),
	)
	auth.RegisterOutboxHandlers(outboxWorker, mailer)
	outboxWorker.Start(ctx)
	defer outboxWorker.Stop()

	// hub := gateway.NewHub(store, cacheMgr, presenceSvc)
	// go hub.Run(ctx)

	authHandler := handler.NewAuth(&authSvc, bind)
	// gatewayHandler := gateway.NewHandler(hub, cacheMgr)
	// userHandler := handler.NewUser(&userSvc, bind)

	app := &Application{
		Config:      cfg,
		RateLimiter: rateLimiter,
		Handlers: Handlers{
			Auth: authHandler,
			// Gateway: gatewayHandler,
			// Me:      meHandler,
			// User:    userHandler,
		},
		// Managers: Managers{
		// 	Token: tokenMgr,
		// },
	}

	return app.Serve(ctx)
}
