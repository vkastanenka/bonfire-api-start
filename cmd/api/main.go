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
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/logger"
	"bonfire-api/internal/pkg/validator"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/relation"
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

	tokenProvider, err := token.NewProvider(token.Config{
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
	bind := httpio.NewBind(val)

	channelCache := cache.NewChannelCache(cacheClient)
	gatewayCache := cache.NewGatewayCache(cacheClient)
	memberCache := cache.NewMemberCache(cacheClient)
	messageCache := cache.NewMessageCache(cacheClient)
	presenceCache := cache.NewPresenceCache(cacheClient)
	relationCache := cache.NewRelationCache(cacheClient)
	sessionCache := cache.NewSessionCache(cacheClient)
	tokenCache := cache.NewTokenCache(cacheClient)
	userCache := cache.NewUserCache(cacheClient)
	wsTicketCache := cache.NewWSTicketCache(cacheClient)

	channelRepo := repository.NewChannelRepository(dbStore)
	memberRepo := repository.NewMemberRepository(dbStore)
	messageRepo := repository.NewMessageRepository(dbStore)
	outboxRepo := repository.NewOutboxRepository(dbStore)
	reactionRepo := repository.NewReactionRepository(dbStore)
	relationRepo := repository.NewRelationRepository(dbStore)
	sessionRepo := repository.NewSessionRepository(dbStore)
	userRepo := repository.NewUserRepository(dbStore)

	cachedChannelRepo := repository.NewCachedChannelRepository(channelCache, channelRepo)
	cachedMemberRepo := repository.NewCachedMemberRepository(memberCache, channelCache, memberRepo)
	cachedMessageRepo := repository.NewCachedMessageRepository(messageCache, messageRepo)
	cachedRelationRepo := repository.NewCachedRelationRepository(relationCache, relationRepo)
	// cachedSessionRepo := repository.NewCachedSessionRepository(sessionCache, sessionRepo)
	cachedUserRepo := repository.NewCachedUserRepository(userCache, userRepo)

	broadcaster := gateway.NewBroadcaster(cacheClient, gatewayCache)

	authSvc := auth.NewService(
		userCache,
		userRepo,
		cachedUserRepo,
		sessionCache,
		sessionRepo,
		outboxRepo,
		wsTicketCache,
		tokenCache,
		tokenProvider,
		dbStore,
	)
	channelSvc := channel.NewChannelService(
		channelCache,
		channelRepo,
		memberCache,
		memberRepo,
		messageCache,
		messageRepo,
		presenceCache,
		cachedUserRepo,
		outboxRepo,
		cachedRelationRepo,
		dbStore,
	)
	gatewaySvc := gateway.NewService(
		broadcaster,
		gatewayCache,
		presenceCache,
		relationCache,
		userCache,
	)
	memberSvc := channel.NewMemberService(
		memberCache,
		memberRepo,
		cachedMemberRepo,
		channelCache,
		channelRepo,
		cachedChannelRepo,
		messageCache,
		messageRepo,
		userCache,
		userRepo,
		cachedUserRepo,
		presenceCache,
		outboxRepo,
		cachedRelationRepo,
		dbStore,
	)
	messageSvc := channel.NewMessageService(
		messageCache,
		messageRepo,
		cachedMessageRepo,
		channelRepo,
		cachedChannelRepo,
		memberCache,
		memberRepo,
		cachedMemberRepo,
		reactionRepo,
		userRepo,
		cachedUserRepo,
		outboxRepo,
		dbStore,
	)
	relationSvc := relation.NewService(
		relationCache,
		relationRepo,
		cachedRelationRepo,
		channelRepo,
		cachedChannelRepo,
		memberRepo,
		cachedMemberRepo,
		outboxRepo,
		presenceCache,
		userCache,
		userRepo,
		cachedUserRepo,
		dbStore,
	)
	sessionSvc := session.NewService(sessionCache, sessionRepo, outboxRepo, dbStore)
	userSvc := user.NewService(userCache, userRepo, cachedUserRepo, cachedRelationRepo, outboxRepo, sessionCache, sessionRepo, dbStore)

	authHandler := handler.NewAuthHandler(authSvc, bind)
	channelHandler := handler.NewChannelHandler(channelSvc, bind)
	healthHandler := handler.NewHealthHandler(dbClient, cacheClient)
	memberHandler := handler.NewMemberHandler(memberSvc, bind)
	messageHandler := handler.NewMessageHandler(messageSvc, bind)
	relationHandler := handler.NewRelationHandler(relationSvc, bind)
	sessionHandler := handler.NewSessionHandler(sessionSvc, bind)
	userHandler := handler.NewUserHandler(userSvc, bind)

	wsHub := gateway.NewHub(cacheClient, gatewaySvc)
	go wsHub.Run(ctx)
	gatewayHandler := gateway.NewHandler(wsHub, wsTicketCache, bind)

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
