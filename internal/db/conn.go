package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnConfig defines the configuration for PostgreSQL connection pool lifecycle management.
type ConnConfig struct {
	ConnString      string
	MaxConns        int
	MinConns        int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	HealthCheck     time.Duration
}

// NewConn initializes a pgxpool client, configures pool settings, and verifies connectivity with a ping.
func NewConn(ctx context.Context, cfg ConnConfig) (*pgxpool.Pool, error) {
	if cfg.ConnString == "" {
		return nil, ErrDBConfigInvalid()
	}

	config, err := pgxpool.ParseConfig(cfg.ConnString)
	if err != nil {
		return nil, ErrDBURLInvalid()
	}

	start := time.Now()

	slog.InfoContext(ctx, "initializing db connection pool",
		slog.String("host", config.ConnConfig.Host),
		slog.Int("port", int(config.ConnConfig.Port)),
		slog.String("database", config.ConnConfig.Database),
	)

	if cfg.MaxConns > 0 {
		config.MaxConns = int32(cfg.MaxConns)
	}
	if cfg.MinConns > 0 {
		config.MinConns = int32(cfg.MinConns)
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
		return nil, ErrDBPoolCreateFailed().
			Meta("host", config.ConnConfig.Host).
			Meta("database", config.ConnConfig.Database).
			Wrap(err)
	}

	pingCtx, cancel := context.WithTimeoutCause(ctx, 5*time.Second, fmt.Errorf("database ping timeout after 5s"))
	defer cancel()

	if err := dbPool.Ping(pingCtx); err != nil {
		dbPool.Close()
		return nil, ErrDBPingFailed().
			Meta("host", config.ConnConfig.Host).
			Meta("database", config.ConnConfig.Database).
			Wrap(err)
	}

	slog.InfoContext(ctx, "db connection established",
		slog.Duration("duration", time.Since(start)),
		slog.String("host", config.ConnConfig.Host),
		slog.String("database", config.ConnConfig.Database),
	)

	return dbPool, nil
}
