package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
func (app *Application) Serve() error {
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
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		slog.Info("shutting down server", "signal", s.String())

		// Give active connections 5 seconds to complete before killing them
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		shutdownError <- srv.Shutdown(ctx)
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
