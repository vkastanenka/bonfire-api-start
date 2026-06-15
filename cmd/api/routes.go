package main

import (
	"net/http"
	"time"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	customMiddleware "bonfire-api/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
)

func (app *Application) routes() http.Handler {
	r := chi.NewRouter()

	// Setup CORS
	corsMiddleware := cors.New(cors.Options{ /* your options */ })

	// Global Middlewares
	r.Use(corsMiddleware.Handler)
	r.Use(middleware.RequestID)
	r.Use(customMiddleware.LoggingMiddleware)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(customMiddleware.SecurityHeaders)

	r.Route("/api/v1", func(api chi.Router) {
		api.Group(func(auth chi.Router) {
			auth.Use(customMiddleware.RateLimit(app.RateLimiter, 5, time.Minute, "auth"))
			// api.Post("/auth/login", httpio.ToHTTP(app.AuthHandler.Login))
			// ... the rest of your routes
		})
	})

	// 404 & 405 Handlers
	r.NotFound(httpio.ToHTTP(func(w http.ResponseWriter, r *http.Request) error {
		return apperr.NewNotFound("The requested API endpoint does not exist.")
	}))

	return r
}
