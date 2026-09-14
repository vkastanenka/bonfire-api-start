package httpio

import (
	"bonfire-api/internal/errs"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-redis/redis_rate/v10"
)

type RateLimitScope string

const (
	RateLimitScopeAuth   RateLimitScope = "auth"
	RateLimitScopePublic RateLimitScope = "public"
	RateLimitScopeAPI    RateLimitScope = "api"
)

type RateLimitConfig struct {
	Limit  int
	Window time.Duration
	Scope  RateLimitScope
}

func RateLimit(limiter *redis_rate.Limiter, cfg RateLimitConfig) func(http.Handler) http.Handler {
	rateLimitConfig := redis_rate.Limit{
		Rate:   cfg.Limit,
		Period: cfg.Window,
		Burst:  cfg.Limit,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			ipAddr, err := CtxGetIP(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "rate limiter missing telemetry context", "error", err)
				ipAddr = extractIP(r, false)
			}
			ip := ipAddr.String()

			redisCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
			defer cancel()

			redisKey := fmt.Sprintf("rl:%s:%s", cfg.Scope, ip)
			res, err := limiter.Allow(redisCtx, redisKey, rateLimitConfig)
			if err != nil {
				slog.WarnContext(ctx, "rate limiter evaluation bypassed (failing open)",
					"error", err,
					"client_ip", ip,
				)
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))

			if res.Allowed == 0 {
				retrySecs := int(res.RetryAfter.Seconds())
				if retrySecs <= 0 {
					retrySecs = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retrySecs))

				rateLimitErr := &errs.Error{
					Code: errs.CodeDeadlineExceeded,
					Err:  fmt.Errorf("rate limit exceeded for ip: %s", ip),
				}

				respondError(w, r, rateLimitErr)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
