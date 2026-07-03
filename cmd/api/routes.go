package main

import (
	"net/http"
	"time"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	customMiddleware "bonfire-api/internal/middleware"

	_ "bonfire-api/docs"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *Application) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(customMiddleware.TelemetryMiddleware)
	r.Use(customMiddleware.LoggingMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(customMiddleware.Cors(app.Config))
	r.Use(customMiddleware.SecurityHeaders)
	r.Use(middleware.Timeout(10 * time.Second))

	r.Route("/api/v1", func(api chi.Router) {
		api.Group(func(publicAuth chi.Router) {
			publicAuth.Use(httpio.RateLimit(app.RateLimiter, httpio.RateLimitConfig{
				Limit:  app.Config.AuthRateLimit,
				Window: app.Config.AuthRateWindow,
				Scope:  httpio.RateLimitScopePublic,
			}))

			publicAuth.Post("/auth/login", httpio.ToHTTP(app.Handlers.Auth.Login))
			publicAuth.Post("/auth/register", httpio.ToHTTP(app.Handlers.Auth.Register))
		})
	})

	r.NotFound(httpio.ToHTTP(func(w http.ResponseWriter, r *http.Request) error {
		return apperr.NewNotFound(nil, "The requested API endpoint does not exist.")
	}))

	r.MethodNotAllowed(httpio.ToHTTP(func(w http.ResponseWriter, r *http.Request) error {
		return apperr.NewMethodNotAllowed(nil, "HTTP method not allowed for this endpoint.")
	}))

	return r
}
