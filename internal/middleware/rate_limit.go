package middleware

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis_rate/v10"
)

// RateLimit middleware factory
func RateLimit(limiter *redis_rate.Limiter, limit int, window time.Duration, keyPrefix string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getIP(r)

			res, err := limiter.Allow(r.Context(), keyPrefix+":"+ip, redis_rate.PerMinute(limit))
			if err != nil {
				log.Printf("Rate limit error: %v", err)
				next.ServeHTTP(w, r)
				return
			}

			if res.Allowed == 0 {
				http.Error(w, "Too many requests. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getIP(r *http.Request) string {
	// 1. Try X-Forwarded-For (standard proxy header)
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		// The first IP is the actual client
		return strings.Split(xForwardedFor, ",")[0]
	}

	// 2. Fallback to RemoteAddr, but strip the port
	ip := strings.Split(r.RemoteAddr, ":")[0]
	return ip
}
