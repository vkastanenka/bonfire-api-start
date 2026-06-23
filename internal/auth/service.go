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

type AuthService struct {
	store        repository.Store
	tokenManager token.Manager
	tokenConfig  TokenConfig
	user         *user.Service
}

func NewAuthService(store repository.Store, tokenManager token.Manager, tokenConfig TokenConfig, user *user.Service) *AuthService {
	return &AuthService{
		store:        store,
		tokenManager: tokenManager,
		tokenConfig:  tokenConfig,
		user:         user,
	}
}
