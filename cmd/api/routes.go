package main

import (
	"net/http"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *Application) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(httpio.CORS(app.Config))
	r.Use(middleware.RequestID)
	r.Use(httpio.Trace)
	r.Use(httpio.ClientTelemetry(app.Config.TrustProxy))
	r.Use(httpio.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(app.Config.RequestTimeout))
	r.Use(httpio.SecurityHeaders)

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

	r.NotFound(httpio.ToHTTP(app.notFoundHandler))
	r.MethodNotAllowed(httpio.ToHTTP(app.methodNotAllowedHandler))

	return r
}

func (app *Application) notFoundHandler(w http.ResponseWriter, r *http.Request) error {
	return apperr.NewNotFound(nil, "The requested API endpoint does not exist.")
}

func (app *Application) methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) error {
	return apperr.NewMethodNotAllowed(nil, "HTTP method not allowed for this endpoint.")
}
