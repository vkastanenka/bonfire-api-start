package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"bonfire-api/internal/auth"
	"bonfire-api/internal/config"
	"bonfire-api/internal/health"

	"github.com/go-redis/redis_rate/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Application dependencies
type Application struct {
	Config        *config.Config
	DB            *pgxpool.Pool
	Redis         *redis.Client
	RateLimiter   *redis_rate.Limiter
	AuthHandler   *auth.AuthHandler
	HealthHandler *health.Handler
}

// Serve configures the HTTP server and manages graceful shutdown.
func (app *Application) Serve(ctx context.Context) error {
	// Init server
	srv := &http.Server{
		Addr:              app.Config.Port,
		Handler:           app.routes(),
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		MaxHeaderBytes:    1 * 1024 * 1024,
		ErrorLog:          slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}

	// Shutdown error channel
	shutdownError := make(chan error)

	// Listen to OS interrupt signals
	go func() {
		// Panic recovery
		defer func() {
			if r := recover(); r != nil {
				slog.Error("recovered from panic in shutdown goroutine", "panic", r)
			}
		}()

		<-ctx.Done() // Block until context has been cancelled
		slog.Info("shutting down core API server")

		// Create a hard cutoff window for active connections to drain
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Set error
		shutdownError <- srv.Shutdown(shutdownCtx)
	}()

	slog.Info("core API server starting", "port", app.Config.Port)

	// Start server
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err // An unexpected error occurred while running
	}

	// Wait for the graceful shutdown to complete
	err = <-shutdownError
	if err != nil {
		return err
	}

	// Finish
	slog.Info("server stopped cleanly")
	return nil
}
