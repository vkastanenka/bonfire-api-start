package auth

import (
	"bonfire-api/internal/repository"
	"bonfire-api/internal/session"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

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
		token:   token,
		session: session,
		user:    user,
	}
}
