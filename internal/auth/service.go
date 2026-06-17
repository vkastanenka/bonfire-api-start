package auth

import (
	"bonfire-api/internal/repository"
	"context"
)

type Store interface {
	repository.Querier
	ExecTx(ctx context.Context, fn func(*repository.Queries) error) error
}

type TokenConfig struct {
	AccessSecret        string
	RefreshSecret       string
	VerificationSecret  string
	PasswordResetSecret string
	PasswordMFASecret   string
}

type AuthService struct {
	store       Store
	tokenConfig TokenConfig
}

func NewAuthService(store Store, tokenConfig TokenConfig) *AuthService {
	return &AuthService{
		store:       store,
		tokenConfig: tokenConfig,
	}
}
