package httpio

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/config"
	"bonfire-api/internal/logger"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-redis/redis_rate/v10"
	"github.com/rs/cors"
)

type contextKey string

const clientMetaKey contextKey = "client_metadata"

type ClientMeta struct {
	IP        netip.Addr
	UserAgent string
}

func ClientTelemetry(trustProxy bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			meta := ClientMeta{
				IP:        extractIP(r, trustProxy),
				UserAgent: r.UserAgent(),
			}

			ctx := context.WithValue(r.Context(), clientMetaKey, meta)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetMeta(ctx context.Context) (ClientMeta, error) {
	meta, ok := ctx.Value(clientMetaKey).(ClientMeta)
	if !ok {
		return ClientMeta{IP: netip.IPv4Unspecified()}, errors.New("client metadata missing from request context")
	}
	return meta, nil
}

func GetIP(ctx context.Context) (netip.Addr, error) {
	meta, err := GetMeta(ctx)
	if err != nil {
		return netip.IPv4Unspecified(), err
	}
	return meta.IP, nil
}

func extractIP(r *http.Request, trustProxy bool) netip.Addr {
	if trustProxy {
		if apiIP := r.Header.Get("CF-Connecting-IP"); apiIP != "" {
			if addr, err := netip.ParseAddr(strings.TrimSpace(apiIP)); err == nil {
				return addr
			}
		}
		if apiIP := r.Header.Get("X-Real-IP"); apiIP != "" {
			if addr, err := netip.ParseAddr(strings.TrimSpace(apiIP)); err == nil {
				return addr
			}
		}

		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if len(parts) > 0 {
				targetIP := strings.TrimSpace(parts[0])
				if addr, err := netip.ParseAddr(targetIP); err == nil {
					return addr
				}
			}
		}
	}

	rawIP := r.RemoteAddr
	if ip, _, err := net.SplitHostPort(rawIP); err == nil {
		rawIP = ip
	}

	rawIP = strings.TrimSuffix(strings.TrimPrefix(rawIP, "["), "]")

	addr, err := netip.ParseAddr(rawIP)
	if err != nil {
		return netip.IPv4Unspecified()
	}

	return addr
}

func CORS(cfg *config.Config) func(http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins: cfg.CORSAllowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-CSRF-Token",
		},
		ExposedHeaders: []string{
			"Link",
			"X-Request-ID",
			"X-Trace-ID",
		},
		AllowCredentials: cfg.CORSAllowCredentials,
		MaxAge:           300,
	})

	return c.Handler
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		slog.InfoContext(r.Context(), "http request processed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"bytes_written", ww.BytesWritten(),
		)
	})
}

func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				if rvr == http.ErrAbortHandler {
					panic(rvr)
				}

				stackTrace := debug.Stack()

				var err error
				if e, ok := rvr.(error); ok {
					err = e
				} else {
					err = fmt.Errorf("%v", rvr)
				}

				appErr := &apperr.Error{
					Code:   apperr.CodeInternal,
					Detail: apperr.CodeInternal.Detail(),
					Err:    err,
				}

				slog.ErrorContext(r.Context(), "catastrophic runtime panic recovered",
					"error.panic_message", err.Error(),
					"error.stack", string(stackTrace),
					"http.method", r.Method,
					"http.path", r.URL.Path,
				)

				respondError(w, r, appErr)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

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

			ipAddr, err := GetIP(ctx)
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

				rateLimitErr := &apperr.Error{
					Code:   apperr.CodeTooManyRequests,
					Detail: apperr.CodeTooManyRequests.Detail(),
					Err:    fmt.Errorf("rate limit exceeded for ip: %s", ip),
				}

				respondError(w, r, rateLimitErr)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Content-Security-Policy", "default-src 'self';")

		next.ServeHTTP(w, r)
	})
}

func Trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if reqID := middleware.GetReqID(ctx); reqID != "" {
			ctx = context.WithValue(ctx, logger.ReqIDKey, reqID)
		}

		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = generateW3CTraceID()
		}
		r.Header.Set("X-Trace-ID", traceID)

		ctx = context.WithValue(ctx, logger.TraceIDKey, traceID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(logger.TraceIDKey).(string); ok {
		return v
	}
	return ""
}

func generateW3CTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
