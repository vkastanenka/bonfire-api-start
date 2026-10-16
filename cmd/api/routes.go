package main

import (
	"net/http"

	"bonfire-api/internal/httpio"
	"bonfire-api/internal/pkg/errs"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *Application) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(httpio.Trace)
	r.Use(httpio.WithClientMeta(app.Config.TrustProxy))
	r.Use(httpio.Logger)
	r.Use(httpio.Recoverer)
	r.Use(httpio.CORS(app.Config))
	r.Use(middleware.Timeout(app.Config.RequestTimeout))
	r.Use(httpio.SecurityHeaders(app.Config.IsProduction()))

	r.Route("/api/v1", func(api chi.Router) {
		api.Group(func(public chi.Router) {
			public.Use(httpio.RateLimit(app.RateLimiter, httpio.RateLimitConfig{
				Limit:  app.Config.PublicRateLimit,
				Window: app.Config.PublicRateLimitWindow,
				Scope:  httpio.RateLimitScopePublic,
			}))

			app.Handlers.Health.RegisterRoutes(public)
			app.Handlers.Auth.RegisterPublicRoutes(public)
			app.Handlers.Gateway.RegisterPublicRoutes(public)
		})

		api.Group(func(auth chi.Router) {
			auth.Use(httpio.RateLimit(app.RateLimiter, httpio.RateLimitConfig{
				Limit:  app.Config.AuthRateLimit,
				Window: app.Config.AuthRateWindow,
				Scope:  httpio.RateLimitScopeAuth,
			}))
			auth.Use(httpio.RequireAuth(app.Tokens))

			app.Handlers.Auth.RegisterAuthenticatedRoutes(auth)
			app.Handlers.User.RegisterRoutes(auth)
			app.Handlers.Gateway.RegisterAuthenticatedRoutes(auth)
		})
	})

	r.NotFound(httpio.ToHTTPErr(app.notFoundHandler))
	r.MethodNotAllowed(httpio.ToHTTPErr(app.methodNotAllowedHandler))

	return r
}

func (app *Application) notFoundHandler(w http.ResponseWriter, r *http.Request) error {
	return errs.NotFound("The requested API endpoint does not exist.")
}

func (app *Application) methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) error {
	return errs.Unimplemented("HTTP method not allowed for this endpoint")
}
