package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/auth"
	"bonfire-api/internal/email-mock"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/validator"
	"bonfire-api/internal/worker"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, falling back to system environment variables")
	}

	// Connect to PostgreSQL
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	dbPool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbPool.Close()

	// Ping connection to confirm it's alive
	if err := dbPool.Ping(ctx); err != nil {
		log.Fatalf("Database ping failed: %v\n", err)
	}

	// 1. Initialize storage and query wrappers
	store := repository.NewStore(dbPool) // For services running atomic transactions
	queries := repository.New(dbPool)    // For the background worker executing raw inline queries

	// 2. Initialize email infrastructure dependencies
	mailer := email.NewLogMockMailer()

	// 3. Initialize Services and pass the store wrapper
	authService := auth.NewAuthService(store)

	// 4. Instantiate and BOOT the background outbox processing engine
	// Polls the database every 1 second, processing up to 10 events per batch
	outboxWorker := worker.NewOutboxWorker(queries, mailer, 1*time.Second, 10)
	outboxWorker.Start()
	defer outboxWorker.Stop() // Guarantees the ticker stops cleanly if main terminates

	// Setup router
	r := chi.NewRouter()

	// Global Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// 4. Initialize validator and handler layer
	val := validator.New()
	authHandler := auth.NewAuthHandler(authService, val)

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/auth/ping", httpio.ToHTTP(authHandler.Ping))
		api.Post("/auth/register", httpio.ToHTTP(authHandler.Register))
	})

	// Chi-idiomatic global fallback for missing endpoints (404)
	r.NotFound(httpio.ToHTTP(func(w http.ResponseWriter, r *http.Request) error {
		return apperr.NewNotFound("The requested API endpoint does not exist.")
	}))

	// Chi-idiomatic global fallback for bad verbs (405)
	r.MethodNotAllowed(httpio.ToHTTP(func(w http.ResponseWriter, r *http.Request) error {
		return apperr.NewMethodNotAllowed("HTTP method not allowed for this endpoint.")
	}))

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Println("Core API Server starting on :8080...")
	log.Fatal(srv.ListenAndServe())
}
