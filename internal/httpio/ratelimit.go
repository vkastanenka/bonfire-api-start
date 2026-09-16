package httpio

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"bonfire-api/internal/pkg/errs"

	"github.com/go-redis/redis_rate/v10"
	"github.com/google/uuid"
)

type RateLimitScope string

const (
	RateLimitScopeAPI    RateLimitScope = "api"
	RateLimitScopeAuth   RateLimitScope = "auth"
	RateLimitScopePublic RateLimitScope = "public"
)

type RateLimitConfig struct {
	Limit  int
	Window time.Duration
	Scope  RateLimitScope
}

// RateLimit creates middleware to restrict incoming request frequency using Redis GCRA.
func RateLimit(limiter *redis_rate.Limiter, cfg RateLimitConfig) func(http.Handler) http.Handler {
	rateLimitConfig := redis_rate.Limit{
		Rate:   cfg.Limit,
		Period: cfg.Window,
		Burst:  cfg.Limit,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Bypass rate limiting safely if Redis/Limiter is uninitialized
			if limiter == nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()

			// Resolve key identifier: prefer Authenticated User ID if present, otherwise IP
			keyID := resolveKeyIdentifier(ctx, r)

			redisCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
			defer cancel()

			redisKey := fmt.Sprintf("rl:%s:%s", cfg.Scope, keyID)
			res, err := limiter.Allow(redisCtx, redisKey, rateLimitConfig)
			if err != nil {
				slog.WarnContext(ctx, "rate limiter evaluation bypassed (failing open)",
					"error", err,
					"scope", cfg.Scope,
					"key_id", keyID,
				)
				next.ServeHTTP(w, r)
				return
			}

			// Populate standard rate limit response headers
			resetTime := time.Now().Add(res.ResetAfter).Unix()
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime, 10))

			if res.Allowed == 0 {
				retrySecs := int(res.RetryAfter.Seconds())
				if retrySecs <= 0 {
					retrySecs = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retrySecs))

				rateLimitErr := errs.ResourceExhausted("Quota exceeded for this endpoint. Please retry after the indicated delay.").
					ErrorInfoReason("RATE_LIMIT_EXCEEDED").
					ErrorInfoMeta("scope", string(cfg.Scope)).
					RetryInfo(res.RetryAfter)

				respondError(w, r, rateLimitErr)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// resolveKeyIdentifier extracts User ID from context if authenticated, defaulting to client IP.
func resolveKeyIdentifier(ctx context.Context, r *http.Request) string {
	if userID, err := CtxGetUserID(ctx); err == nil && userID != uuid.Nil {
		return "user:" + userID.String()
	}

	if ipAddr, err := CtxGetIP(ctx); err == nil {
		return "ip:" + ipAddr.String()
	}

	return "ip:" + extractIP(r, false).String()
}
