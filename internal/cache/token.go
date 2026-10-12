package cache

import (
	"context"
	"math"
	"time"

	"bonfire-api/internal/pkg/errs"
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
	return c.consumeJTI(ctx, "forgot_password", claims)
}

func (c *TokenCache) ConsumeRefresh(ctx context.Context, claims *token.Claims) error {
	return c.consumeJTI(ctx, "refresh", claims)
}

func (c *TokenCache) ConsumeEmailVerify(ctx context.Context, claims *token.Claims) error {
	return c.consumeJTI(ctx, "email_verify", claims)
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

func (c *TokenCache) consumeJTI(ctx context.Context, keyCategory string, claims *token.Claims) error {
	if claims == nil {
		return errs.Internal("token claims cannot be nil").
			Resource("type", keyCategory, "", "")
	}

	remainingTTL := time.Until(claims.ExpiresAt.Time) + token.DefaultClockLeeway
	if remainingTTL <= 0 {
		return ErrTokenAlreadyUsed()
	}

	ttlSeconds := int64(math.Ceil(remainingTTL.Seconds()))
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}

	key := "token:consumed:" + keyCategory + ":{" + claims.ID + "}"

	res, err := tokenConsumeJTIScript.Run(
		ctx,
		c.client,
		[]string{key},
		ttlSeconds,
	).Int64()

	if err != nil {
		return redis.NewError(err, redis.ResourceToken)
	}

	if res == 0 {
		return ErrTokenAlreadyUsed().Meta("type", keyCategory)
	}

	return nil
}
