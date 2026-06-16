package middleware

import (
	"bonfire-api/internal/config"
	"context"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-redis/redis_rate/v10"
	"github.com/rs/cors"
)

type contextKey string

const loggerKey contextKey = "logger"

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Example CSP: Only allow resources from your own domain
		w.Header().Set("Content-Security-Policy", "default-src 'self';")

		next.ServeHTTP(w, r)
	})
}

// RateLimit middleware factory
func RateLimit(limiter *redis_rate.Limiter, limit int, window time.Duration, keyPrefix string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use the getIP helper to get the real client IP
			ip := getIP(r)

			res, err := limiter.Allow(r.Context(), keyPrefix+":"+ip, redis_rate.PerMinute(limit))
			if err != nil {
				// Fail open: log the error and allow the request
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

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := middleware.GetReqID(r.Context())

		// Add request_id to the logger context
		logger := slog.Default().With("request_id", requestID)

		// Attach the logger to the request context
		ctx := context.WithValue(r.Context(), loggerKey, logger)

		logger.Info("started request", "method", r.Method, "path", r.URL.Path)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
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

func NewCors(cfg *config.Config) *cors.Cors {
	return cors.New(cors.Options{
		AllowedOrigins: strings.Split(cfg.CORSAllowedOrigins, ","),
		AllowedMethods: []string{
			http.MethodGet, http.MethodPost, http.MethodPut,
			http.MethodDelete, http.MethodOptions,
		},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: cfg.CORSAllowCredentials,
		MaxAge:           300,
	})
}
