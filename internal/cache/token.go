package cache

import (
	"context"
	"errors"
	"time"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/token"

	redisdriver "github.com/redis/go-redis/v9"
)

func ErrTokenAlreadyUsed() error {
	return errs.InvalidArgument("Token already used.").
		FieldViolation("token", "Token already used.", "INVALID").
		Wrap(errors.New("valid token is required"))
}

type TokenCache struct {
	client redisdriver.Cmdable
}

func NewTokenCache(client redisdriver.Cmdable) *TokenCache {
	return &TokenCache{client: client}
}

func tokenConsumedForgotPasswordKey(jti string) string {
	return "token:consumed:forgot-password:" + jti
}

// Lua script to atomically check if a JTI is consumed and mark it in a single network round trip.
var tokenConsumeJTIScript = redisdriver.NewScript(`
	local key = KEYS[1]
	local ttl = tonumber(ARGV[1])

	-- If the key exists, it has already been used
	if redis.call('EXISTS', key) == 1 then
		return 0 -- already consumed
	end

	-- Mark as consumed with the remaining token TTL
	redis.call('SET', key, '1', 'EX', ttl)
	return 1 -- successfully consumed
`)

// ConsumeForgotPasswordJTI checks if the JTI was used, and if not, marks it as used atomically.
// Returns true if the token was successfully consumed, false if it was already used.
func (c *TokenCache) ConsumeForgotPasswordJTI(ctx context.Context, jti string, remainingTTL time.Duration) (bool, error) {
	if remainingTTL <= 0 {
		return false, nil
	}

	ttlSeconds := int(remainingTTL.Seconds())
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}

	res, err := tokenConsumeJTIScript.Run(
		ctx,
		c.client,
		[]string{tokenConsumedForgotPasswordKey(jti)},
		ttlSeconds,
	).Int64()

	if err != nil {
		return false, redis.NewError(err, redis.ScopeToken)
	}

	return res == 1, nil
}

func (c *TokenCache) ConsumePasswordResetToken(ctx context.Context, claims *token.Claims) error {
	remainingTTL := time.Until(claims.ExpiresAt.Time)

	consumed, err := c.ConsumeForgotPasswordJTI(ctx, claims.ID, remainingTTL)
	if err != nil {
		return err
	}
	if !consumed {
		return ErrTokenAlreadyUsed()
	}
	return nil
}
