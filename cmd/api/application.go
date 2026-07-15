package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"bonfire-api/internal/auth"
	"bonfire-api/internal/config"
	"bonfire-api/internal/gateway"
	"bonfire-api/internal/me"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"

	"github.com/go-redis/redis_rate/v10"
)

type Handlers struct {
	Auth    *auth.Handler
	Gateway *gateway.Handler
	Me      *me.Handler
	User    *user.Handler
}

type Managers struct {
	Token *token.Manager
}

type Application struct {
	Config      *config.Config
	RateLimiter *redis_rate.Limiter
	Handlers    Handlers
	Managers    Managers
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
