package auth

import (
	"bonfire-api/internal/cache"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/session"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"
	"context"
	"log/slog"
	"time"

	"golang.org/x/sync/singleflight"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type Service struct {
	store       repository.Store
	cache       cache.Store
	token       *token.Manager
	session     *session.Service
	user        *user.Service
	flightGroup singleflight.Group
}

func NewService(
	store repository.Store,
	cache cache.Store,
	token *token.Manager,
	session *session.Service,
	user *user.Service,
) *Service {
	return &Service{
		store:       store,
		cache:       cache,
		token:       token,
		session:     session,
		user:        user,
		flightGroup: singleflight.Group{},
	}
}

func (s *Service) createCacheSession(ctx context.Context, auth session.AuthView) {
	sessionKey := cache.SessionKey(auth.ID)

	if err := s.cache.Set(ctx, sessionKey, auth, time.Until(auth.ExpiresAt)); err != nil {
		slog.ErrorContext(ctx, "failed to cache new session",
			"error", err,
			"id", auth.ID,
			"user_id", auth.UserID,
		)
	}
}

func (s *Service) updateCacheSession(ctx context.Context, auth session.AuthView) {
	sessionKey := cache.SessionKey(auth.ID)

	if err := s.cache.Set(ctx, sessionKey, auth, time.Until(auth.ExpiresAt)); err != nil {
		slog.ErrorContext(ctx, "failed to update cached session",
			"error", err,
			"id", auth.ID,
			"user_id", auth.UserID,
		)

		if delErr := s.cache.Delete(ctx, sessionKey); delErr != nil {
			slog.WarnContext(ctx, "failed to evict stale session from cache after update failure",
				"error", delErr,
				"id", auth.ID,
				"user_id", auth.UserID,
			)
		}
	}
}
