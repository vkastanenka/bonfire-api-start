package shield

import (
	"bonfire-api/internal/redis"
	"context"
	"time"

	redisdriver "github.com/redis/go-redis/v9"
)

const (
	defaultMaxAttempts = 5
	defaultFailuresTTL = 1 * time.Minute
	defaultCooldownTTL = 1 * time.Minute
)

func cacheIdentifier(identifier string) string {
	return "{" + identifier + "}"
}

func FailuresKey(domain, action, identifier string) string {
	return "shield:failures:" + domain + ":" + action + ":" + cacheIdentifier(identifier)
}

func CooldownKey(domain, action, identifier string) string {
	return "shield:cooldown:" + domain + ":" + action + ":" + cacheIdentifier(identifier)
}

type ShieldCache struct {
	rdb redisdriver.Cmdable
}

func NewShieldCache(rdb redisdriver.Cmdable) *ShieldCache {
	return &ShieldCache{rdb: rdb}
}

func (c *ShieldCache) GetCooldown(ctx context.Context, domain, action, identifier string) (bool, time.Duration, error) {
	k := CooldownKey(domain, action, identifier)

	ttl, err := c.rdb.TTL(ctx, k).Result()
	if err != nil {
		if redis.IsCacheMiss(err) {
			return false, 0, nil
		}
		return false, 0, redis.NewError(err, redis.ResourceShield)
	}

	if ttl <= 0 {
		return false, 0, nil
	}

	return true, ttl, nil
}

func (c *ShieldCache) SetCooldown(ctx context.Context, domain, action, identifier string) error {
	k := CooldownKey(domain, action, identifier)

	err := c.rdb.Set(ctx, k, "1", defaultCooldownTTL).Err()
	if err != nil {
		return redis.NewError(err, redis.ResourceShield)
	}

	return nil
}

var recordFailureScript = redisdriver.NewScript(`
	local fKey = KEYS[1]
	local cKey = KEYS[2]

	local maxAttempts = tonumber(ARGV[1])
	local failuresTTL = tonumber(ARGV[2])
	local cooldownTTL = tonumber(ARGV[3])

	-- 1. Increment failure count and refresh rolling window TTL
	local count = redis.call('INCR', fKey)
	redis.call('EXPIRE', fKey, failuresTTL)

	-- 2. If max attempts reached, trigger the cooldown automatically
	if count >= maxAttempts then
		redis.call('SET', cKey, '1', 'EX', cooldownTTL)
	end

	return count
`)

func (c *ShieldCache) RecordFailure(ctx context.Context, domain, action, identifier string) (int, error) {
	fKey := FailuresKey(domain, action, identifier)
	cKey := CooldownKey(domain, action, identifier)

	keys := []string{fKey, cKey}
	args := []any{
		defaultMaxAttempts,
		int(defaultFailuresTTL.Seconds()),
		int(defaultCooldownTTL.Seconds()),
	}

	count, err := recordFailureScript.Run(ctx, c.rdb, keys, args...).Int64()
	if err != nil {
		return 0, redis.NewError(err, redis.ResourceShield)
	}

	return int(count), nil
}

func (c *ShieldCache) Clear(ctx context.Context, domain, action, identifier string) error {
	cKey := CooldownKey(domain, action, identifier)
	fKey := FailuresKey(domain, action, identifier)

	_, err := c.rdb.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.Del(ctx, cKey)
		pipe.Del(ctx, fKey)
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ResourceShield)
	}

	return nil
}
