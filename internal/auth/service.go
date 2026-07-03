package auth

import (
	"bonfire-api/internal/repository"
	"bonfire-api/internal/session"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"
)

type Service struct {
	store   repository.Store
	session *session.Service
	token   *token.Manager
	user    *user.Service
}

func NewService(
	store repository.Store,
	session *session.Service,
	token *token.Manager,
	user *user.Service,
) *Service {
	return &Service{
		store:   store,
		session: session,
		token:   token,
		user:    user,
	}
}
