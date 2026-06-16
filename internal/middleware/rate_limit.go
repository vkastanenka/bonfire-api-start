package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis_rate/v10"
)

// RateLimit middleware factory
func RateLimit(limiter *redis_rate.Limiter, limit int, window time.Duration, keyPrefix string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIP(r)
			ctx := r.Context()

			// Init config
			rateLimitConfig := redis_rate.Limit{
				Rate:   limit,
				Period: window,
				Burst:  limit,
			}

			res, err := limiter.Allow(ctx, keyPrefix+":"+ip, rateLimitConfig)
			if err != nil {
				slog.ErrorContext(ctx, "rate limit evaluation failed", "error", err, "client_ip", ip)

				// Fail open: prioritize availability over strict blocking if Redis stumbles
				next.ServeHTTP(w, r)
				return
			}

			// Inject headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))

			if res.Allowed == 0 {
				// Inject Retry-After header
				retrySecs := int(res.RetryAfter.Seconds())
				if retrySecs <= 0 {
					retrySecs = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retrySecs))

				http.Error(w, "Too many requests. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getIP(r *http.Request) string {
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		return strings.TrimSpace(strings.Split(xForwardedFor, ",")[0])
	}

	// Handles both IPv4 and IPv6 formatting
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
