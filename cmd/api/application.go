package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"bonfire-api/internal/config"
	"bonfire-api/internal/gateway"
	"bonfire-api/internal/handler"
	"bonfire-api/internal/token"

	"github.com/go-redis/redis_rate/v10"
)

type Handlers struct {
	Auth    *handler.Auth
	Channel *handler.Channel
	Gateway *gateway.Handler
	Health  *handler.Health
	Me      *handler.Me
	Outbox  *handler.Outbox
	User    *handler.User
}

type Application struct {
	Config      *config.Config
	RateLimiter *redis_rate.Limiter
	Tokens      *token.Provider
	Handlers    Handlers
}

func (app *Application) Serve(ctx context.Context) error {
	srv := &http.Server{
		Addr:              app.Config.Port,
		Handler:           app.routes(),
		ReadTimeout:       app.Config.ServerReadTimeout,
		WriteTimeout:      app.Config.ServerWriteTimeout,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		MaxHeaderBytes:    1 * 1024 * 1024,
		ErrorLog:          slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}

	shutdownError := make(chan error)
	serverDone := make(chan struct{})

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("recovered from panic in shutdown goroutine", "panic", r)
			}
		}()

		select {
		case <-ctx.Done():
			slog.Info("shutting down core API server")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), app.Config.ShutdownTimeout)
			defer cancel()
			shutdownError <- srv.Shutdown(shutdownCtx)
		case <-serverDone:
			return
		}
	}()

	slog.Info("core API server starting", "port", app.Config.Port)

	err := srv.ListenAndServe()
	close(serverDone)

	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError
	if err != nil {
		return err
	}

	slog.Info("server stopped cleanly")
	return nil
}
