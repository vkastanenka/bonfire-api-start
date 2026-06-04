package main

import (
	"log"
	"net/http"

	"bonfire-api/internal/handler"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()

	// Essential Middlewares
	r.Use(middleware.Logger)    // Logs all incoming requests
	r.Use(middleware.Recoverer) // Recovers from panics without crashing the server

	// Initialize Handlers
	userHandler := &handler.UserHandler{}

	// Route Definitions
	r.Route("/api/v1", func(api chi.Router) {
		api.Post("/auth/register", userHandler.Register)
		api.Post("/auth/login", userHandler.Login)
		
		// Example of a CRUD sub-route
		api.Get("/users/{id}", userHandler.GetProfile)
	})

	log.Println("Core API Server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}