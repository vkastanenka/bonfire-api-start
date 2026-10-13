package auth

import (
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"
	"context"
	"time"

	"github.com/google/uuid"
)

type Worker interface {
	RegisterHandler(eventType string, handler outbox.Handler)
}

type Broadcaster interface {
	BroadcastToUser(ctx context.Context, recipientID uuid.UUID, eventType string, payload any, excludeSessionIDs ...uuid.UUID) error
	BroadcastToUserSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, eventType string, payload any) error
}

type Mailer interface {
	SendPasswordReset(ctx context.Context, emailAddress string, resetToken string, requestedAt time.Time) error
	SendRegister(ctx context.Context, emailAddress string, username string, token string, requestedAt time.Time) error
	SendResendVerification(ctx context.Context, emailAddress string, username string, token string, requestedAt time.Time) error
}

type ShieldCache interface {
	Clear(ctx context.Context, domain string, action string, identifier string) error
	GetCooldown(ctx context.Context, domain string, action string, identifier string) (bool, time.Duration, error)
	RecordFailure(ctx context.Context, domain string, action string, identifier string) (int, error)
	SetCooldown(ctx context.Context, domain string, action string, identifier string) error
}

type TokenCache interface {
	ConsumeEmailVerify(ctx context.Context, claims *token.Claims) error
	ConsumePasswordReset(ctx context.Context, claims *token.Claims) error
	ConsumeRefresh(ctx context.Context, claims *token.Claims) error
}

type UserCache interface {
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteBatch(ctx context.Context, ids []uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*user.User, []uuid.UUID, error)
	Set(ctx context.Context, usr *user.User) error
	SetBatch(ctx context.Context, users map[uuid.UUID]*user.User) error
}

type OutboxRepository interface {
	Create(ctx context.Context, e *outbox.Event) error
}

type UserRepository interface {
	Create(ctx context.Context, u *user.User) (*user.User, error)
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) ([]*user.User, error)
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	GetByUsername(ctx context.Context, username string) (*user.User, error)
	ListDeleteScheduled(ctx context.Context, currentTime time.Time, limitVal int) ([]*user.User, error)
	SetDeleteSchedule(ctx context.Context, id uuid.UUID, deleteScheduledAt *time.Time, disabledAt *time.Time, updatedAt time.Time) (*user.User, error)
	SetDisabled(ctx context.Context, id uuid.UUID, disabledAt *time.Time, updatedAt time.Time) (*user.User, error)
	Update(ctx context.Context, u *user.User) (*user.User, error)
	UpdateBatch(ctx context.Context, users []*user.User) ([]*user.User, error)
	UpdateEmail(ctx context.Context, id uuid.UUID, email string, updatedAt time.Time) (*user.User, error)
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string, updatedAt time.Time) (*user.User, error)
	UpdatePhone(ctx context.Context, id uuid.UUID, phone *string, updatedAt time.Time) (*user.User, error)
	UpdatePreferredPresence(ctx context.Context, id uuid.UUID, preferredPresence *presence.Presence, preferredPresenceUntil *time.Time, updatedAt time.Time) (*user.User, error)
	UpdateProfile(ctx context.Context, id uuid.UUID, displayName string, bio *string, avatarURL *string, bannerColor *string, updatedAt time.Time) (*user.User, error)
	UpdateUsername(ctx context.Context, id uuid.UUID, username string, updatedAt time.Time) (*user.User, error)
	Verify(ctx context.Context, id uuid.UUID, verifiedAt *time.Time, updatedAt time.Time) (*user.User, error)
}

type CachedUserRepository interface {
	Get(ctx context.Context, id uuid.UUID) (user.User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]user.User, error)
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}

type TokenProvider interface {
	GenerateAccess(uid uuid.UUID, sid uuid.UUID) (string, time.Time, error)
	GenerateEmailVerify(userID uuid.UUID) (string, time.Time, error)
	GeneratePair(uid uuid.UUID, sid uuid.UUID) (token.Pair, error)
	GeneratePasswordReset(userID uuid.UUID) (string, time.Time, error)
	GenerateRefresh(uid uuid.UUID, sid uuid.UUID) (string, time.Time, error)
	VerifyAccess(tokenStr string) (*token.Claims, error)
	VerifyEmailVerify(tokenStr string) (*token.Claims, error)
	VerifyPasswordReset(tokenStr string) (*token.Claims, error)
	VerifyRefresh(tokenStr string) (*token.Claims, error)
}
