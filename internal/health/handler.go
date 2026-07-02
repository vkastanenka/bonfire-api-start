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

// Handler
type Handler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

// NewHandler
func NewHandler(db *pgxpool.Pool, redis *redis.Client) *Handler {
	return &Handler{
		db:    db,
		redis: redis,
	}
}

// Check performs validation
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) error {
	// 2 second max deadline
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Verify PostgreSQL Connectivity
	if err := h.db.Ping(ctx); err != nil {
		return apperr.NewInternal(err)
	}

	// Verify Redis Connectivity
	if err := h.redis.Ping(ctx).Err(); err != nil {
		return apperr.NewInternal(err)
	}

	httpio.RespondOK(w, r, struct{}{}, "Healthy.")
	return nil
}
