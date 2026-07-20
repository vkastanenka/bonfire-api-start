package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ConnConfig struct {
	ConnString      string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	HealthCheck     time.Duration
}

func NewConn(ctx context.Context, cfg ConnConfig) (*pgxpool.Pool, error) {
	if cfg.ConnString == "" {
		return nil, fmt.Errorf("db connection string cannot be empty")
	}

	start := time.Now()
	slog.Info("initializing db connection")

	config, err := pgxpool.ParseConfig(cfg.ConnString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse db config: %w", err)
	}

	if cfg.MaxConns > 0 {
		config.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		config.MinConns = cfg.MinConns
	}
	if cfg.MaxConnLifetime > 0 {
		config.MaxConnLifetime = cfg.MaxConnLifetime
	}
	if cfg.MaxConnIdleTime > 0 {
		config.MaxConnIdleTime = cfg.MaxConnIdleTime
	}
	if cfg.HealthCheck > 0 {
		config.HealthCheckPeriod = cfg.HealthCheck
	}

	dbPool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create db connection: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := dbPool.Ping(pingCtx); err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("db connection verification failed: %w", err)
	}

	slog.Info("db connection established", slog.Duration("duration", time.Since(start)))
	return dbPool, nil
}
