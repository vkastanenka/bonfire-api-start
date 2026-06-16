package auth

import (
	"bonfire-api/internal/repository"
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Store interface {
	ValidateUserCredentialsAvailability(ctx context.Context, arg repository.ValidateUserCredentialsAvailabilityParams) (repository.ValidateUserCredentialsAvailabilityRow, error)
	CreateOutboxEvent(ctx context.Context, arg repository.CreateOutboxEventParams) (repository.CreateOutboxEventRow, error)
	CreateSession(ctx context.Context, arg repository.CreateSessionParams) (repository.CreateSessionRow, error)
	CreateUser(ctx context.Context, arg repository.CreateUserParams) (repository.CreateUserRow, error)
	GetUserByEmail(ctx context.Context, email string) (repository.GetUserByEmailRow, error)
	GetUserAuthCredentials(ctx context.Context, email string) (repository.GetUserAuthCredentialsRow, error)
	UpdateSessionRefreshToken(ctx context.Context, arg repository.UpdateSessionRefreshTokenParams) error
	UpdateUserPassword(ctx context.Context, arg repository.UpdateUserPasswordParams) error
	VerifyUserEmail(ctx context.Context, arg repository.VerifyUserEmailParams) error
	EnableUserTOTP(ctx context.Context, arg repository.EnableUserTOTPParams) error
	DisableUserTOTP(ctx context.Context, id pgtype.UUID) error
	GetUserTOTPSecret(ctx context.Context, id pgtype.UUID) (pgtype.Text, error)

	// Sessions & Device Management
	GetSession(ctx context.Context, refreshToken string) (repository.GetSessionRow, error) // <--- ADD THIS BACK
	GetUserSessions(ctx context.Context, userID pgtype.UUID) ([]repository.GetUserSessionsRow, error)
	DeleteSession(ctx context.Context, arg repository.DeleteSessionParams) error
	DeleteAllSessionsExcept(ctx context.Context, arg repository.DeleteAllSessionsExceptParams) error
	ExecTx(ctx context.Context, fn func(*repository.Queries) error) error
}

type UserProvider interface {
	GetUserByID(ctx context.Context, userID uuid.UUID) (repository.GetUserByIDRow, error)
}

type TokenConfig struct {
	AccessSecret        string
	RefreshSecret       string
	VerificationSecret  string
	PasswordResetSecret string
}

type AuthService struct {
	store        Store
	userProvider UserProvider
	tokenConfig  TokenConfig
}

func NewAuthService(store Store, userProvider UserProvider, tokenConfig TokenConfig) *AuthService {
	return &AuthService{
		store:        store,
		userProvider: userProvider,
		tokenConfig:  tokenConfig,
	}
}
