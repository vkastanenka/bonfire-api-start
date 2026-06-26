package auth

import (
	"bonfire-api/internal/cache"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/session"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"
)

type Service struct {
	store   repository.Store
	cache   cache.Store
	session *session.Service
	token   *token.Service
	user    *user.Service
}

func NewService(
	store repository.Store,
	cache cache.Store,
	session *session.Service,
	token *token.Service,
	user *user.Service,
) *Service {
	return &Service{
		store: store,
		cache: cache,
		session: session,
		token: token,
		user:  user,
	}
}
