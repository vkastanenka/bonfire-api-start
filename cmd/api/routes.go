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
	r.Use(middleware.RequestID)
	r.Use(customMiddleware.LoggingMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(customMiddleware.Cors(app.Config))
	r.Use(customMiddleware.SecurityHeaders)
	r.Use(middleware.Timeout(15 * time.Second))

	// Swagger docs
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Health check
	r.Get("/healthz", app.Handlers.Health.HealthCheck)

	// Routes
	r.Route("/api/v1", func(api chi.Router) {
		// Public routes
		api.Group(func(publicAuth chi.Router) {
			publicAuth.Use(customMiddleware.RateLimit(app.RateLimiter, 5, time.Minute, "auth"))

			// Testing
			publicAuth.Get("/user/ping", httpio.ToHTTP(app.Handlers.User.Ping))
			publicAuth.Get("/user/{id}", httpio.ToHTTP(app.Handlers.User.GetByID))
			publicAuth.Get("/user", httpio.ToHTTP(app.Handlers.User.GetByEmail))
			publicAuth.Delete("/user", httpio.ToHTTP(app.Handlers.User.DeleteByEmail))

			publicAuth.Get("/user_profile/ping", httpio.ToHTTP(app.Handlers.UserProfile.Ping))
			publicAuth.Get("/user_profile/{id}", httpio.ToHTTP(app.Handlers.UserProfile.GetByUserID))

			publicAuth.Get("/outbox_events/ping", httpio.ToHTTP(app.Handlers.OutboxEvents.Ping))
			publicAuth.Get("/outbox_events/count", httpio.ToHTTP(app.Handlers.OutboxEvents.Count))
			publicAuth.Post("/outbox_events", httpio.ToHTTP(app.Handlers.OutboxEvents.Create))
			publicAuth.Get("/outbox_events", httpio.ToHTTP(app.Handlers.OutboxEvents.List))
			publicAuth.Get("/outbox_events/{id}", httpio.ToHTTP(app.Handlers.OutboxEvents.GetByID))
			publicAuth.Put("/outbox_events/{id}", httpio.ToHTTP(app.Handlers.OutboxEvents.UpdateByID))
			publicAuth.Delete("/outbox_events/{id}", httpio.ToHTTP(app.Handlers.OutboxEvents.DeleteByID))

			publicAuth.Get("/auth/ping", httpio.ToHTTP(app.Handlers.Auth.Ping))
			publicAuth.Post("/auth/register", httpio.ToHTTP(app.Handlers.Auth.Register))
			publicAuth.Post("/auth/verify", httpio.ToHTTP(app.Handlers.Auth.VerifyEmail))
			publicAuth.Post("/auth/resend-verification-email", httpio.ToHTTP(app.Handlers.Auth.ResendVerificationEmail))
			publicAuth.Post("/auth/login", httpio.ToHTTP(app.Handlers.Auth.Login))
			publicAuth.Post("/auth/refresh", httpio.ToHTTP(app.Handlers.Auth.Refresh))
			publicAuth.Post("/auth/forgot-password", httpio.ToHTTP(app.Handlers.Auth.ForgotPassword))
			publicAuth.Post("/auth/reset-password", httpio.ToHTTP(app.Handlers.Auth.ResetPassword))
			publicAuth.Post("/auth/login/2fa", httpio.ToHTTP(app.Handlers.Auth.VerifyLogin2FA))
		})

		// Protected routes
		api.Group(func(protected chi.Router) {
			protected.Use(customMiddleware.RateLimit(app.RateLimiter, 100, time.Minute, "api"))
			protected.Use(auth.RequireAuth(app.TokenManager, app.Config.AccessSecret))

			protected.Get("/auth/devices", httpio.ToHTTP(app.Handlers.Auth.GetDevices))
			protected.Delete("/auth/devices", httpio.ToHTTP(app.Handlers.Auth.RevokeAllOtherDevices))
			protected.Delete("/auth/devices/{id}", httpio.ToHTTP(app.Handlers.Auth.RevokeDevice))

			// Require verification routes
			protected.Group(func(verified chi.Router) {
				verified.Use(auth.RequireVerified())

				verified.Post("/users/me/2fa/generate", httpio.ToHTTP(app.Handlers.Auth.GenerateTOTP))
				verified.Post("/users/me/2fa/enable", httpio.ToHTTP(app.Handlers.Auth.EnableTOTP))
			})
		})
	})

	// Global fallback for missing endpoints (404)
	r.NotFound(httpio.ToHTTP(func(w http.ResponseWriter, r *http.Request) error {
		return apperr.New(apperr.CodeNotFound, "The requested API endpoint does not exist.")
	}))

	// Global fallback for bad verbs (405)
	r.MethodNotAllowed(httpio.ToHTTP(func(w http.ResponseWriter, r *http.Request) error {
		return apperr.New(apperr.CodeMethodNotAllowed, "HTTP method not allowed for this endpoint.")
	}))

	return r
}
