package main

import (
	"net/http"
	"time"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/auth"
	"bonfire-api/internal/httpio"
	customMiddleware "bonfire-api/internal/middleware"

	_ "bonfire-api/docs"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func (app *Application) routes() http.Handler {
	// Init router
	r := chi.NewRouter()

	// Global middleware
	r.Use(customMiddleware.NewCors(app.Config).Handler)
	r.Use(middleware.RequestID)
	r.Use(customMiddleware.LoggingMiddleware)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(customMiddleware.SecurityHeaders)
	r.Use(middleware.Timeout(15 * time.Second))

	// Swagger docs
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Health check
	r.Get("/healthz", app.HealthHandler.HealthCheck)

	// Routes
	r.Route("/api/v1", func(api chi.Router) {
		// Public routes
		api.Group(func(publicAuth chi.Router) {
			publicAuth.Use(customMiddleware.RateLimit(app.RateLimiter, 5, time.Minute, "auth"))

			publicAuth.Get("/auth/ping", httpio.ToHTTP(app.AuthHandler.Ping))
			publicAuth.Post("/auth/register", httpio.ToHTTP(app.AuthHandler.Register))
			publicAuth.Post("/auth/verify", httpio.ToHTTP(app.AuthHandler.VerifyEmail))
			publicAuth.Post("/auth/resend-verification-email", httpio.ToHTTP(app.AuthHandler.ResendVerificationEmail))
			publicAuth.Post("/auth/login", httpio.ToHTTP(app.AuthHandler.Login))
			publicAuth.Post("/auth/refresh", httpio.ToHTTP(app.AuthHandler.RefreshToken))
			publicAuth.Post("/auth/forgot-password", httpio.ToHTTP(app.AuthHandler.ForgotPassword))
			publicAuth.Post("/auth/reset-password", httpio.ToHTTP(app.AuthHandler.ResetPassword))
			publicAuth.Post("/auth/login/2fa", httpio.ToHTTP(app.AuthHandler.VerifyLogin2FA))
		})

		// Protected routes
		api.Group(func(protected chi.Router) {
			protected.Use(customMiddleware.RateLimit(app.RateLimiter, 100, time.Minute, "api"))
			protected.Use(auth.RequireAuth(app.Config.AccessSecret))

			protected.Get("/auth/devices", httpio.ToHTTP(app.AuthHandler.GetDevices))
			protected.Delete("/auth/devices", httpio.ToHTTP(app.AuthHandler.RevokeAllOtherDevices))
			protected.Delete("/auth/devices/{id}", httpio.ToHTTP(app.AuthHandler.RevokeDevice))

			// Require verification routes
			protected.Group(func(verified chi.Router) {
				verified.Use(auth.RequireVerified())

				verified.Post("/users/me/2fa/generate", httpio.ToHTTP(app.AuthHandler.GenerateTOTP))
				verified.Post("/users/me/2fa/enable", httpio.ToHTTP(app.AuthHandler.EnableTOTP))
			})
		})
	})

	// Global fallback for missing endpoints (404)
	r.NotFound(httpio.ToHTTP(func(w http.ResponseWriter, r *http.Request) error {
		return apperr.NewNotFound("The requested API endpoint does not exist.")
	}))

	// Global fallback for bad verbs (405)
	r.MethodNotAllowed(httpio.ToHTTP(func(w http.ResponseWriter, r *http.Request) error {
		return apperr.NewMethodNotAllowed("HTTP method not allowed for this endpoint.")
	}))

	return r
}
