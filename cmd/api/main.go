package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"bonfire-api/internal/auth"
	"bonfire-api/internal/cache"
	"bonfire-api/internal/channel"
	"bonfire-api/internal/config"
	"bonfire-api/internal/db"
	"bonfire-api/internal/gateway"
	"bonfire-api/internal/handler"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/logger"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/relation"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/session"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"
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

	cacheConn, err := redis.NewConn(ctx, redis.ConnConfig{
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

	tokenProvider, err := token.NewProvider(token.Config{
		Issuer: cfg.TokenIssuer,
		Access: token.TypeConfig{
			Secret: cfg.AccessSecret,
			TTL:    cfg.JWTAccessTTL,
		},
		Refresh: token.TypeConfig{
			Secret: cfg.RefreshSecret,
			TTL:    cfg.JWTRefreshTTL,
		},
		EmailVerify: token.TypeConfig{
			Secret: cfg.EmailVerifySecret,
			TTL:    cfg.JWTEmailVerifyTTL,
		},
		PasswordReset: token.TypeConfig{
			Secret: cfg.PasswordResetSecret,
			TTL:    cfg.JWTPasswordResetTTL,
		},
	})
	if err != nil {
		return err
	}
	dbStore := db.NewStore(dbConn)
	rateLimiter := redis_rate.NewLimiter(cacheConn)
	val := validator.New()
	bind := httpio.NewBind(val)

	ticketCache := cache.NewWSTicketCache(cacheConn, redis.ScopeTicket, cfg.TicketTTL)
	userCache := cache.NewUserCache(cacheConn, redis.ScopeUser, cfg.UserTTL)

	channelRepo := repository.NewChannelRepository(dbStore)
	memberRepo := repository.NewMemberRepository(dbStore)
	messageRepo := repository.NewMessageRepository(dbStore, memberRepo)
	outboxRepo := repository.NewOutboxRepository(dbStore)
	reactionRepo := repository.NewReactionRepository(dbStore)
	relationRepo := repository.NewRelationRepository(dbStore)
	sessionRepo := repository.NewSessionRepository(dbStore)
	userRepo := repository.NewUserRepository(dbStore)

	authSvc := auth.NewService(
		userRepo,
		sessionRepo,
		outboxRepo,
		ticketCache,
		tokenProvider,
		dbStore,
	)
	channelSvc := channel.NewChannelService(
		channelRepo,
		memberRepo,
		messageRepo,
		reactionRepo,
		userRepo,
		userCache,
		outboxRepo,
		relationRepo,
		dbStore,
	)
	memberSvc := channel.NewMemberService(
		memberRepo,
		channelRepo,
		messageRepo,
		userRepo,
		userCache,
		outboxRepo,
		relationRepo,
		dbStore,
	)
	messageSvc := channel.NewMessageService(
		messageRepo,
		channelRepo,
		memberRepo,
		reactionRepo,
		userRepo,
		userCache,
		outboxRepo,
		dbStore,
	)
	relationSvc := relation.NewService(
		relationRepo,
		userRepo,
		userCache,
		channelRepo,
		memberRepo,
		outboxRepo,
		dbStore,
	)
	sessionSvc := session.NewService(sessionRepo, outboxRepo, dbStore)
	userSvc := user.NewService(userRepo, userCache, outboxRepo, dbStore)

	authHandler := handler.NewAuthHandler(authSvc, bind)
	channelHandler := handler.NewChannelHandler(channelSvc, bind)
	healthHandler := handler.NewHealthHandler(dbConn, cacheConn)
	memberHandler := handler.NewMemberHandler(memberSvc, bind)
	messageHandler := handler.NewMessageHandler(messageSvc, bind)
	relationHandler := handler.NewRelationHandler(relationSvc, bind)
	sessionHandler := handler.NewSessionHandler(sessionSvc, bind)
	userHandler := handler.NewUserHandler(userSvc, relationSvc, channelSvc, bind)

	wsHub := gateway.NewHub(dbStore, cacheConn)
	go wsHub.Run(ctx)
	gatewayHandler := gateway.NewHandler(wsHub, userCache, ticketCache, bind)
	// broadcaster := gateway.NewBroadcaster(cacheConn)

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

	outboxWorker.Start(ctx)
	defer outboxWorker.Stop()

	app := &Application{
		Config:      cfg,
		RateLimiter: rateLimiter,
		Tokens:      tokenProvider,
		Handlers: Handlers{
			Auth:     authHandler,
			Channel:  channelHandler,
			Gateway:  gatewayHandler,
			Health:   healthHandler,
			Member:   memberHandler,
			Message:  messageHandler,
			Relation: relationHandler,
			Session:  sessionHandler,
			User:     userHandler,
		},
	}

	return app.Serve(ctx)
}
