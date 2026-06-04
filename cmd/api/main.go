package main

import (
	"context"
	"log"
	"net/http"

	"bonfire-api/internal/handler"
	"bonfire-api/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()
	
	// 1. Connect to PostgreSQL
	connStr := "postgres://postgres:password123@localhost:5432/discord_db?sslmode=disable"
	dbPool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbPool.Close()

	// Ping connection to confirm it's alive
	if err := dbPool.Ping(ctx); err != nil {
		log.Fatalf("Database ping failed: %v\n", err)
	}

	// 2. Initialize sqlc Queries
	queries := repository.New(dbPool)

	// 3. Setup router and inject our DB queries into the handler
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	userHandler := &handler.UserHandler{
		DB: queries, // Injecting the database access layer
	}

	r.Route("/api/v1", func(api chi.Router) {
		api.Post("/auth/register", userHandler.Register)
		api.Get("/users/{id}", userHandler.GetProfile)
	})

	log.Println("Core API Server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}