package handler

import (
	"bonfire-api/internal/auth"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"

	"github.com/google/uuid"
)

type AuthService interface {
	ForgotPassword(ctx context.Context, email string) error
	Login(ctx context.Context, p auth.LoginParams) (auth.LoginResult, error)
	Refresh(ctx context.Context, r auth.RefreshParams) (auth.RefreshResult, error)
	Register(ctx context.Context, p auth.RegisterParams) (auth.RegisterResult, error)
	ResendVerify(ctx context.Context, userID uuid.UUID) error
	ResetPassword(ctx context.Context, p auth.ResetPasswordParams) (auth.ResetPasswordResult, error)
	VerifyEmail(ctx context.Context, tokenStr string) error
	WSTicket(ctx context.Context, uid uuid.UUID) (uuid.UUID, error)
}

type UserService interface {
	CheckAvailability(ctx context.Context, email user.Email, username user.Username) (emailAvail bool, usernameAvail bool, err error)
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetByEmail(ctx context.Context, email user.Email) (*user.User, error)
	GetByUsername(ctx context.Context, username user.Username) (*user.User, error)
	SetPreferredPresence(ctx context.Context, id uuid.UUID, p *presence.Presence) (*user.User, error)
	UpdateProfile(ctx context.Context, p user.UpdateProfileParams) (*user.User, error)
	VerifyAccount(ctx context.Context, id uuid.UUID) error
}
