package health

import (
	"context"
	"net/http"
	"time"

	"bonfire-api/internal/httpio"
	"bonfire-api/internal/pkg/errs"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

type Cache interface {
	Ping(context.Context) *redis.StatusCmd
}

type Store interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	cache Cache
	store Store
}

func NewHandler(cache Cache, store Store) *Handler {
	return &Handler{
		cache: cache,
		store: store,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/healthz", httpio.ToHTTPErr(h.check))
}

func (h *Handler) check(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.cache.Ping(ctx).Err(); err != nil {
		return errs.Internal("Cache health check failed.").Wrap(err)
	}

	if err := h.store.Ping(ctx); err != nil {
		return errs.Internal("Database health check failed.").Wrap(err)
	}

	httpio.RespondNoContent(w)
	return nil
}
