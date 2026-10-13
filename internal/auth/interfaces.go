package auth

import (
	"context"
	"time"

	"bonfire-api/internal/outbox"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

type Broadcaster interface {
	BroadcastToUser(ctx context.Context, recipientID uuid.UUID, eventType string, payload any, excludeSessionIDs ...uuid.UUID) error
	BroadcastToUserSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, eventType string, payload any) error
}

type CachedUserRepository interface {
	Get(ctx context.Context, id uuid.UUID) (user.User, error)
}

type Mailer interface {
	SendPasswordReset(ctx context.Context, emailAddress string, username string, resetToken string, requestedAt time.Time) error
	SendRegister(ctx context.Context, emailAddress string, username string, token string, requestedAt time.Time) error
	SendResendVerification(ctx context.Context, emailAddress string, username string, token string, requestedAt time.Time) error
}

type OutboxRepository interface {
	Create(ctx context.Context, e *outbox.Event) error
}

type ShieldCache interface {
	GetCooldown(ctx context.Context, domain string, action string, identifier string) (bool, time.Duration, error)
	RecordFailure(ctx context.Context, domain string, action string, identifier string) (int, error)
	SetCooldown(ctx context.Context, domain string, action string, identifier string) error
}

type TokenCache interface {
	ConsumeEmailVerify(ctx context.Context, claims *token.Claims) error
	ConsumePasswordReset(ctx context.Context, claims *token.Claims) error
	ConsumeRefresh(ctx context.Context, claims *token.Claims) error
}

type TokenProvider interface {
	GenerateEmailVerify(userID uuid.UUID) (string, time.Time, error)
	GeneratePair(uid uuid.UUID, sid uuid.UUID) (token.Pair, error)
	GeneratePasswordReset(userID uuid.UUID) (string, time.Time, error)
	VerifyEmailVerify(tokenStr string) (*token.Claims, error)
	VerifyPasswordReset(tokenStr string) (*token.Claims, error)
	VerifyRefresh(tokenStr string) (*token.Claims, error)
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}

type UserCache interface {
	Delete(ctx context.Context, id uuid.UUID) error
}

type UserRepository interface {
	Create(ctx context.Context, u *user.User) (*user.User, error)
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	SetDeleteSchedule(ctx context.Context, id uuid.UUID, deleteScheduledAt *time.Time, disabledAt *time.Time, updatedAt time.Time) (*user.User, error)
	SetDisabled(ctx context.Context, id uuid.UUID, disabledAt *time.Time, updatedAt time.Time) (*user.User, error)
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string, updatedAt time.Time) (*user.User, error)
	Verify(ctx context.Context, id uuid.UUID, verifiedAt *time.Time, updatedAt time.Time) (*user.User, error)
}

type Worker interface {
	RegisterHandler(eventType string, handler outbox.Handler)
}
