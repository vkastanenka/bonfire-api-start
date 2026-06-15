package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"bonfire-api/internal/auth"
	"bonfire-api/internal/config"

	"github.com/go-redis/redis_rate/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Application holds the application-wide dependencies.
type Application struct {
	Config      *config.Config
	DB          *pgxpool.Pool
	Redis       *redis.Client
	RateLimiter *redis_rate.Limiter
	AuthHandler *auth.AuthHandler
}

// Serve configures the HTTP server and manages graceful shutdown.
func (app *Application) Serve(ctx context.Context) error {
	srv := &http.Server{
		Addr:         app.Config.Port,
		Handler:      app.routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Create a channel to catch shutdown errors from our background goroutine
	shutdownError := make(chan error)

	// Run a background goroutine to listen for OS interrupt signals
	go func() {
		<-ctx.Done() // Blocks until SIGINT or SIGTERM is received
		slog.Info("shutting down core API server")

		// Create a hard cutoff window for active connections to drain
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		shutdownError <- srv.Shutdown(shutdownCtx)
	}()

	slog.Info("Core API Server starting", "port", app.Config.Port)

	// This blocks until the server is shut down
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err // An unexpected error occurred while running
	}

	// Wait for the graceful shutdown to complete
	err = <-shutdownError
	if err != nil {
		return err
	}

	slog.Info("server stopped cleanly")
	return nil
}
