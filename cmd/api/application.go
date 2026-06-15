package main

import (
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
