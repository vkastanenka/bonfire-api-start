package health

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// --- HANDLER CONSTANTS ---

// Errors
const (
	ErrDBCheck    = "Database connection check failed."
	ErrRedisCheck = "Redis connection check failed."
)

// --- HANDLER TYPES ---

// Handler
type Handler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

// --- HANDLER INITIALIZATION ---

// NewHandler
func NewHandler(db *pgxpool.Pool, redis *redis.Client) *Handler {
	return &Handler{
		db:    db,
		redis: redis,
	}
}

// --- HANDLER METHODS ---

// Check performs validation
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) error {
	// 2 second max deadline
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Verify PostgreSQL Connectivity
	if err := h.db.Ping(ctx); err != nil {
		return apperr.New(
			apperr.CodeInternal,
			ErrDBCheck,
			apperr.WithErr(err),
		)
	}

	// Verify Redis Connectivity
	if err := h.redis.Ping(ctx).Err(); err != nil {
		return apperr.New(
			apperr.CodeInternal,
			ErrRedisCheck,
			apperr.WithErr(err),
		)
	}

	httpio.RespondOK(w, r, struct{}{}, "Healthy.")
	return nil
}
