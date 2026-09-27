package cache

import (
	"context"
	"math"
	"time"

	"bonfire-api/internal/redis"
	"bonfire-api/internal/token"

	redisdriver "github.com/redis/go-redis/v9"
)

type TokenCache struct {
	client redisdriver.Cmdable
}

func NewTokenCache(client redisdriver.Cmdable) *TokenCache {
	return &TokenCache{client: client}
}

func (c *TokenCache) ConsumePasswordReset(ctx context.Context, claims *token.Claims) error {
	return c.consumeJTI(ctx, "forgot_password", claims.ID, claims.ExpiresAt.Time)
}

func (c *TokenCache) ConsumeRefresh(ctx context.Context, claims *token.Claims) error {
	return c.consumeJTI(ctx, "refresh", claims.ID, claims.ExpiresAt.Time)
}

func (c *TokenCache) ConsumeEmailVerify(ctx context.Context, claims *token.Claims) error {
	return c.consumeJTI(ctx, "email_verify", claims.ID, claims.ExpiresAt.Time)
}

var tokenConsumeJTIScript = redisdriver.NewScript(`
	local key = KEYS[1]
	local ttl = tonumber(ARGV[1])

	if redis.call('EXISTS', key) == 1 then
		return 0
	end

	redis.call('SET', key, '1', 'EX', ttl)
	return 1
`)

func (c *TokenCache) consumeJTI(ctx context.Context, keyCategory, jti string, expiresAt time.Time) error {
	remainingTTL := time.Until(expiresAt) + token.DefaultClockLeeway
	if remainingTTL <= 0 {
		return ErrTokenAlreadyUsed()
	}

	ttlSeconds := int64(math.Ceil(remainingTTL.Seconds()))
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}

	key := "token:consumed:" + keyCategory + ":" + jti

	res, err := tokenConsumeJTIScript.Run(
		ctx,
		c.client,
		[]string{key},
		ttlSeconds,
	).Int64()

	if err != nil {
		return redis.NewError(err, redis.ScopeToken)
	}

	if res == 0 {
		return ErrTokenAlreadyUsed().Meta("type", keyCategory)
	}

	return nil
}
