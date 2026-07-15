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

	r.Use(middleware.RequestID)
	r.Use(httpio.Trace)
	r.Use(httpio.WithClientMeta(app.Config.TrustProxy))
	r.Use(httpio.Logger)
	r.Use(httpio.Recoverer)
	r.Use(httpio.CORS(app.Config))
	r.Use(middleware.Timeout(app.Config.RequestTimeout))
	r.Use(httpio.SecurityHeaders)

	r.Route("/api/v1", func(api chi.Router) {
		api.Group(func(public chi.Router) {
			// public.Use(httpio.RateLimit(app.RateLimiter, httpio.RateLimitConfig{
			// 	Limit:  app.Config.AuthRateLimit,
			// 	Window: app.Config.AuthRateWindow,
			// 	Scope:  httpio.RateLimitScopePublic,
			// }))

			public.Post("/auth/register", httpio.ToHTTPErr(app.Handlers.Auth.Register))
			public.Post("/auth/login", httpio.ToHTTPErr(app.Handlers.Auth.Login))
			public.Post("/auth/refresh", httpio.ToHTTPErr(app.Handlers.Auth.Refresh))
			public.Post("/auth/verify-email", httpio.ToHTTPErr(app.Handlers.Auth.VerifyEmail))
			public.Get("/gateway/ws", httpio.ToHTTPErr(app.Handlers.Gateway.ServeWS))

			public.Get("/users", httpio.ToHTTPErr(app.Handlers.User.Get))
			public.Get("/users/{id}", httpio.ToHTTPErr(app.Handlers.User.GetByID))
		})

		api.Group(func(auth chi.Router) {
			// auth.Use(httpio.RateLimit(app.RateLimiter, httpio.RateLimitConfig{
			// 	Limit:  app.Config.AuthRateLimit,
			// 	Window: app.Config.AuthRateWindow,
			// 	Scope:  httpio.RateLimitScopeAuth,
			// }))
			auth.Use(httpio.RequireAuth(app.Managers.Token))

			auth.Post("/auth/ws-ticket", httpio.ToHTTPErr(app.Handlers.Auth.WSTicket))
			auth.Post("/auth/resend-verify", httpio.ToHTTPErr(app.Handlers.Auth.ResendVerify))
			auth.Get("/users/@me", httpio.ToHTTPErr(app.Handlers.Me.GetByID))
		})
	})

	r.NotFound(httpio.ToHTTPErr(app.notFoundHandler))
	r.MethodNotAllowed(httpio.ToHTTPErr(app.methodNotAllowedHandler))

	return r
}

func (app *Application) notFoundHandler(w http.ResponseWriter, r *http.Request) error {
	return apperr.NewNotFound(nil, "The requested API endpoint does not exist.")
}

func (app *Application) methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) error {
	return apperr.NewMethodNotAllowed(nil, "HTTP method not allowed for this endpoint.")
}
