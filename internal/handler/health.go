package handler

import (
	"context"
	"net/http"
	"time"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/httpio"

	"github.com/redis/go-redis/v9"
)

type HealthStore interface {
	Ping(ctx context.Context) error
}

type HealthCache interface {
	Ping(context.Context) *redis.StatusCmd
}

type HealthHandler struct {
	store HealthStore
	cache HealthCache
}

func NewHealth(store HealthStore, cache HealthCache) *HealthHandler {
	return &HealthHandler{
		store: store,
		cache: cache,
	}
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.store.Ping(ctx); err != nil {
		return errs.Internal("Database health check failed.").Wrap(err)
	}

	if err := h.cache.Ping(ctx).Err(); err != nil {
		return errs.Internal("Cache health check failed.").Wrap(err)
	}

	httpio.RespondNoContent(w)
	return nil
}
