package auth

import (
	"bonfire-api/internal/repository"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"
	"bonfire-api/internal/userprofile"
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
	userProfile  *userprofile.Service
}

func NewAuthService(store repository.Store, tokenManager token.Manager, tokenConfig TokenConfig, user *user.Service, userprofile *userprofile.Service) *AuthService {
	return &AuthService{
		store:        store,
		tokenManager: tokenManager,
		tokenConfig:  tokenConfig,
		user:         user,
		userProfile:  userprofile,
	}
}
