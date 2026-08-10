package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Querier interface {
	// Basic String & Key Operations
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	MGet(ctx context.Context, keys ...string) ([]interface{}, error)
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
	Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)

	// Hashes
	HSet(ctx context.Context, key string, field string, value interface{}) error
	HGet(ctx context.Context, key, field string, dest interface{}) error
	HDel(ctx context.Context, key string, fields ...string) error
	HGetAll(ctx context.Context, key string, dest *map[string]string) error

	// Sorted Sets (ZSET)
	ZAdd(ctx context.Context, key string, score float64, member interface{}) error
	ZRem(ctx context.Context, key string, members ...interface{}) error
	ZRevRangeByScoreWithScores(ctx context.Context, key string, max, min string, offset, count int64) ([]redis.Z, error)
	ZRemRangeByRank(ctx context.Context, key string, start, stop int64) error

	// Key Expiry & PubSub
	Expire(ctx context.Context, key string, ttl time.Duration) error
	Publish(ctx context.Context, channel string, payload interface{}) error
}

var _ Querier = (*Queries)(nil)
