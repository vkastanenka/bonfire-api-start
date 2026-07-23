package handler

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/httpio"
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHandler(db *pgxpool.Pool, redis *redis.Client) *Handler {
	return &Handler{
		db:    db,
		redis: redis,
	}
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		return errs.Internal("").Wrap(err)
	}

	if err := h.redis.Ping(ctx).Err(); err != nil {
		return errs.Internal("").Wrap(err)
	}

	httpio.RespondNoContent(w)
	return nil
}
