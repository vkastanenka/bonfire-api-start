package auth

import "bonfire-api/internal/repository"

type TokenConfig struct {
	AccessSecret        string
	RefreshSecret       string
	VerificationSecret  string
	PasswordResetSecret string
	PasswordMFASecret   string
}

type AuthService struct {
	store       repository.Store
	tokenConfig TokenConfig
}

func NewAuthService(store repository.Store, tokenConfig TokenConfig) *AuthService {
	return &AuthService{
		store:       store,
		tokenConfig: tokenConfig,
	}
}
