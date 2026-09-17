package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"bonfire-api/internal/pkg/errs"

	goredis "github.com/redis/go-redis/v9"
)

type ConnConfig struct {
	ConnString      string
	PoolSize        int
	MinIdleConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
	DialTimeout     time.Duration
}

// NewConn initializes a go-redis client, configures pool settings, and verifies connectivity with a PING.
func NewConn(ctx context.Context, cfg ConnConfig) (*goredis.Client, error) {
	if cfg.ConnString == "" {
		return nil, errs.InvalidArgument("redis connection string cannot be empty").
			Reason("REDIS_CONFIG_INVALID").
			FieldViolation("conn_string", "connection string is required", "REQUIRED")
	}

	opt, err := goredis.ParseURL(cfg.ConnString)
	if err != nil {
		return nil, errs.InvalidArgument("invalid redis url").
			Reason("REDIS_URL_INVALID").
			FieldViolation("conn_string", "failed to parse redis connection URL", "INVALID_FORMAT").
			Wrap(err)
	}

	start := time.Now()
	slog.InfoContext(ctx, "initializing redis client pool", slog.String("addr", opt.Addr))

	if cfg.PoolSize > 0 {
		opt.PoolSize = cfg.PoolSize
	}
	if cfg.MinIdleConns > 0 {
		opt.MinIdleConns = cfg.MinIdleConns
	}
	if cfg.ConnMaxIdleTime > 0 {
		opt.ConnMaxIdleTime = cfg.ConnMaxIdleTime
	}
	if cfg.ConnMaxLifetime > 0 {
		opt.ConnMaxLifetime = cfg.ConnMaxLifetime
	}
	if cfg.DialTimeout > 0 {
		opt.DialTimeout = cfg.DialTimeout
	}

	rdb := goredis.NewClient(opt)

	pingCtx, cancel := context.WithTimeoutCause(ctx, 5*time.Second, fmt.Errorf("redis ping timeout after 5s"))
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		_ = rdb.Close()
		return nil, errs.Unavailable("redis connection verification failed").
			Reason("REDIS_PING_FAILED").
			Meta("address", opt.Addr).
			Wrap(err)
	}

	slog.InfoContext(ctx, "redis connection established",
		slog.Duration("duration", time.Since(start)),
		slog.String("addr", opt.Addr),
	)

	return rdb, nil
}
