package db

import (
	"bonfire-api/internal/pkg/errs"
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

// NewConn initializes a pgxpool client, configures pool settings, and verifies connectivity with a PING.
func NewConn(ctx context.Context, cfg ConnConfig) (*pgxpool.Pool, error) {
	if cfg.ConnString == "" {
		return nil, errs.InvalidArgument("db connection string cannot be empty").
			Reason("DB_CONFIG_INVALID").
			FieldViolation("conn_string", "connection string is required", "REQUIRED")
	}

	config, err := pgxpool.ParseConfig(cfg.ConnString)
	if err != nil {
		return nil, errs.InvalidArgument("invalid db connection string").
			Reason("DB_URL_INVALID").
			FieldViolation("conn_string", "failed to parse database connection string", "INVALID_FORMAT").
			Wrap(err)
	}

	start := time.Now()
	slog.InfoContext(ctx, "initializing db connection pool",
		slog.String("host", config.ConnConfig.Host),
		slog.Int("port", int(config.ConnConfig.Port)),
		slog.String("database", config.ConnConfig.Database),
	)

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
		return nil, errs.Unavailable("failed to create db pool").
			Reason("DB_POOL_CREATE_FAILED").
			Meta("host", config.ConnConfig.Host).
			Meta("database", config.ConnConfig.Database).
			Wrap(err)
	}

	pingCtx, cancel := context.WithTimeoutCause(ctx, 5*time.Second, fmt.Errorf("db ping timeout after 5s"))
	defer cancel()

	if err := dbPool.Ping(pingCtx); err != nil {
		dbPool.Close()
		return nil, errs.Unavailable("db connection verification failed").
			Reason("DB_PING_FAILED").
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
