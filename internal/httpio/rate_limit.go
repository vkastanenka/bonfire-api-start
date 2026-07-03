package httpio

import (
	"bonfire-api/internal/apperr"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis_rate/v10"
)

// RateLimitScope defines redis keyspaces.
type RateLimitScope string

const (
	RateLimitScopeAuth   RateLimitScope = "auth"
	RateLimitScopePublic RateLimitScope = "public"
	RateLimitScopeAPI    RateLimitScope = "api"
)

func (s RateLimitScope) String() string {
	return string(s)
}

// RateLimitConfig holds the config for the rate limiter.
type RateLimitConfig struct {
	Limit  int
	Window time.Duration
	Scope  RateLimitScope
}

// RateLimit returns a standard HTTP middleware handler for req throttling.
func RateLimit(limiter *redis_rate.Limiter, cfg RateLimitConfig) func(http.Handler) http.Handler {
	// Setup cfg
	rateLimitConfig := redis_rate.Limit{
		Rate:   cfg.Limit,
		Period: cfg.Window,
		Burst:  cfg.Limit,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIP(r)
			ctx := r.Context()

			// Get redis ctx
			redisCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
			defer cancel()

			// Build redis key
			redisKey := fmt.Sprintf("rl:%s:%s", cfg.Scope, ip)
			res, err := limiter.Allow(redisCtx, redisKey, rateLimitConfig)

			if err != nil {
				// Fail Open
				slog.WarnContext(ctx, "rate limiter evaluation bypassed (failing open)",
					"error", err,
					"client_ip", ip,
				)
				next.ServeHTTP(w, r)
				return
			}

			// Standard Tracking Headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))

			// Handle Rate Limit Exhaustion
			if res.Allowed == 0 {
				retrySecs := int(res.RetryAfter.Seconds())
				if retrySecs <= 0 {
					retrySecs = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retrySecs))

				// Construct error.
				rateLimitErr := &apperr.Error{
					Code:   apperr.CodeTooManyRequests,
					Detail: apperr.CodeTooManyRequests.Detail(),
					Err:    fmt.Errorf("rate limit exceeded for ip: %s", ip),
				}

				RespondError(w, r, rateLimitErr)
				return
			}

			// Serve next
			next.ServeHTTP(w, r)
		})
	}
}
