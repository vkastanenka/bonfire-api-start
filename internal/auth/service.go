package auth

import (
	"bonfire-api/internal/redis"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/session"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"
)

type Service struct {
	store   repository.Store
	redis   redis.Store
	session *session.Service
	token   *token.Service
	user    *user.Service
}

func NewService(
	store repository.Store,
	redis redis.Store,
	session *session.Service,
	token *token.Service,
	user *user.Service,
) *Service {
	return &Service{
		store:   store,
		redis:   redis,
		session: session,
		token:   token,
		user:    user,
	}
}
