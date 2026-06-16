package health

import (
	"bonfire-api/internal/httpio"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Handler holds the specific dependencies needed just for health checks.
type Handler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

// NewHandler creates a new health handler.
func NewHandler(db *pgxpool.Pool, redis *redis.Client) *Handler {
	return &Handler{
		db:    db,
		redis: redis,
	}
}

// Check performs the actual system health validation.
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(r.Context()); err != nil {
		httpio.RespondTextError(w, r, "db health check failed", err, http.StatusInternalServerError, "DATABASE_UNREACHABLE")
		return
	}

	if err := h.redis.Ping(r.Context()).Err(); err != nil {
		httpio.RespondTextError(w, r, "redis health check failed", err, http.StatusInternalServerError, "REDIS_UNREACHABLE")
		return
	}

	httpio.RespondText(w, http.StatusOK, "OK")
}
