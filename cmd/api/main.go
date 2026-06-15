package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
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

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, falling back to system environment variables")
	}

	// 1. Load Configurations and Enforce Constraints
	cfg := config.Load()

	// 2. Establish Infrastructure Connection Pools
	dbPool, err := database.NewPostgresPool(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Critical Error: Database initialization failed: %v\n", err)
	}
	defer dbPool.Close()

	rdb, err := cache.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Critical Error: Redis initialization failed: %v\n", err)
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

	// 6. Instantiate Background Outbox System Threads
	outboxWorker := worker.NewOutboxWorker(queries, mailer, 1*time.Second, 10)
	outboxWorker.Start()
	defer outboxWorker.Stop()

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

	// 9. Boot and Configure the Base HTTP Production Server Layer
	srv := &http.Server{
		Addr:         cfg.Port,
		Handler:      app.routes(), // Dispatches handling directly into routes.go
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("Core API Server starting on %s...\n", cfg.Port)
	log.Fatal(srv.ListenAndServe())
}

func initLogger() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}
