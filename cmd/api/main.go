package main

import (
	// "context"
	"log"
	"net/http"
	"time"

	// "os"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/auth"
	"bonfire-api/internal/httpio"

	// "bonfire-api/internal/handler"
	// "bonfire-api/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	// "github.com/jackc/pgx/v5/pgxpool"
	// "github.com/joho/godotenv"
)

func main() {
	// ctx := context.Background()

	// if err := godotenv.Load(); err != nil {
	// 	log.Println("No .env file found, falling back to system environment variables")
	// }

	// // 1. Connect to PostgreSQL
	// connStr := os.Getenv("DATABASE_URL")
	// if connStr == "" {
	// 	log.Fatal("DATABASE_URL environment variable is required")
	// }

	// dbPool, err := pgxpool.New(ctx, connStr)
	// if err != nil {
	// 	log.Fatalf("Unable to connect to database: %v\n", err)
	// }
	// defer dbPool.Close()

	// // Ping connection to confirm it's alive
	// if err := dbPool.Ping(ctx); err != nil {
	// 	log.Fatalf("Database ping failed: %v\n", err)
	// }

	// 2. Initialize sqlc Queries
	// queries := repository.New(dbPool)

	// userHandler := &handler.UserHandler{
	// 	DB: queries, // Injecting the database access layer
	// }

	// 3. Setup router and inject our DB queries into the handler
	r := chi.NewRouter()

	// Global Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	authHandler := &auth.AuthHandler{}

	r.Route("/api/v1", func(api chi.Router) {
		// Unified developer experience: everything uses httpio.ToHTTP
		api.Get("/auth/ping", httpio.ToHTTP(authHandler.Ping))
		api.Post("/auth/register", httpio.ToHTTP(authHandler.Register))
	})

	// Chi-idiomatic global fallback for missing endpoints (404)
	r.NotFound(httpio.ToHTTP(func(w http.ResponseWriter, r *http.Request) error {
		return apperr.NewNotFound("The requested API endpoint does not exist.")
	}))

	// Chi-idiomatic global fallback for bad verbs (405)
	r.MethodNotAllowed(httpio.ToHTTP(func(w http.ResponseWriter, r *http.Request) error {
		// Note: Ensure your apperr mapper returns http.StatusMethodNotAllowed for this type!
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
