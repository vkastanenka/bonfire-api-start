package main

import (
	"log/slog"
	"net/http"
	"time"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/auth"
	"bonfire-api/internal/httpio"
	bfMiddleware "bonfire-api/internal/middleware"

	_ "bonfire-api/docs"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func (app *Application) routes() http.Handler {
	r := chi.NewRouter()

	// swagger docs
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// ----------------------------------------------------
	// INFRASTRUCTURE HEALTH CHECKS (Un-throttled & Public)
	// ----------------------------------------------------
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		// Verify Postgres connection pool is healthy
		if err := app.DB.Ping(r.Context()); err != nil {
			slog.ErrorContext(r.Context(), "health check failed: database unreachable", "error", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("DATABASE_UNREACHABLE"))
			return
		}

		// Verify Redis connection is healthy
		if err := app.Redis.Ping(r.Context()).Err(); err != nil {
			slog.ErrorContext(r.Context(), "health check failed: redis unreachable", "error", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("REDIS_UNREACHABLE"))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// TODO: Move variables to config
	// Initialize CORS from original setup configuration
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "https://yourproductiondomain.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	// Global Middlewares
	r.Use(corsMiddleware.Handler)
	r.Use(middleware.RequestID)
	r.Use(bfMiddleware.LoggingMiddleware)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(bfMiddleware.SecurityHeaders)
	r.Use(middleware.Timeout(15 * time.Second))

	r.Route("/api/v1", func(api chi.Router) {
		// ----------------------------------------------------
		// PUBLIC ROUTES (No token required)
		// ----------------------------------------------------
		api.Group(func(publicAuth chi.Router) {
			publicAuth.Use(bfMiddleware.RateLimit(app.RateLimiter, 5, time.Minute, "auth"))

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

		// ----------------------------------------------------
		// PROTECTED ROUTES (Valid Access Token required)
		// ----------------------------------------------------
		api.Group(func(protected chi.Router) {
			protected.Use(bfMiddleware.RateLimit(app.RateLimiter, 100, time.Minute, "api"))
			protected.Use(auth.RequireAuth(app.Config.AccessSecret))

			protected.Get("/auth/devices", httpio.ToHTTP(app.AuthHandler.GetDevices))
			protected.Delete("/auth/devices", httpio.ToHTTP(app.AuthHandler.RevokeAllOtherDevices))
			protected.Delete("/auth/devices/{id}", httpio.ToHTTP(app.AuthHandler.RevokeDevice))

			// Strict Verification Sub-Group
			protected.Group(func(verified chi.Router) {
				verified.Use(auth.RequireVerified())

				verified.Post("/users/me/2fa/generate", httpio.ToHTTP(app.AuthHandler.GenerateTOTP))
				verified.Post("/users/me/2fa/enable", httpio.ToHTTP(app.AuthHandler.EnableTOTP))
			})
		})
	})

	// Chi-idiomatic global fallback for missing endpoints (404)
	r.NotFound(httpio.ToHTTP(func(w http.ResponseWriter, r *http.Request) error {
		return apperr.NewNotFound("The requested API endpoint does not exist.")
	}))

	// Chi-idiomatic global fallback for bad verbs (405)
	r.MethodNotAllowed(httpio.ToHTTP(func(w http.ResponseWriter, r *http.Request) error {
		return apperr.NewMethodNotAllowed("HTTP method not allowed for this endpoint.")
	}))

	return r
}
