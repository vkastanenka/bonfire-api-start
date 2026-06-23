package auth

import (
	"bonfire-api/internal/repository"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"
)

type TokenConfig struct {
	AccessSecret        string
	RefreshSecret       string
	VerificationSecret  string
	PasswordResetSecret string
	PasswordMFASecret   string
}

type Service struct {
	store        repository.Store
	tokenManager token.Manager
	tokenConfig  TokenConfig
	user         *user.Service
}

func NewService(store repository.Store, tokenManager token.Manager, tokenConfig TokenConfig, user *user.Service) *Service {
	return &Service{
		store:        store,
		tokenManager: tokenManager,
		tokenConfig:  tokenConfig,
		user:         user,
	}
}
