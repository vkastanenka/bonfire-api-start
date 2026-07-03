package auth

import (
	"bonfire-api/internal/repository"
	"bonfire-api/internal/session"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"
)

type Service struct {
	store   repository.Store
	token   *token.Manager
	session *session.Service
	user    *user.Service
}

func NewService(
	store repository.Store,
	token *token.Manager,
	session *session.Service,
	user *user.Service,
) *Service {
	return &Service{
		store:   store,
		session: session,
		token:   token,
		user:    user,
	}
}
