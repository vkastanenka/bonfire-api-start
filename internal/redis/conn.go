package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type ConnConfig struct {
	ConnString      string
	PoolSize        int
	MinIdleConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
}

func NewConn(ctx context.Context, cfg ConnConfig) (*goredis.Client, error) {
	if cfg.ConnString == "" {
		return nil, fmt.Errorf("cache connection string cannot be empty")
	}

	start := time.Now()
	slog.Info("initializing cache client pool")

	opt, err := goredis.ParseURL(cfg.ConnString)
	if err != nil {
		return nil, fmt.Errorf("invalid cache url: %w", err)
	}

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

	rdb := goredis.NewClient(opt)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("cache connection verification failed: %w", err)
	}

	slog.Info("cache connection established", slog.Duration("duration", time.Since(start)))
	return rdb, nil
}
