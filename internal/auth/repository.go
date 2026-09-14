package auth

import (
	"bonfire-api/internal/presence"
	"bonfire-api/internal/session"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"
	"context"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

type OutboxRepository interface {
	Publish(ctx context.Context, eventType string, payload any, now time.Time) error
}

type SessionRepository interface {
	Create(ctx context.Context, s *session.Session) (*session.Session, error)
	DeleteBatchExpired(ctx context.Context, now time.Time, limitVal int) error
	Get(ctx context.Context, id uuid.UUID) (*session.Session, error)
	ListValidByUserID(ctx context.Context, userID uuid.UUID, now time.Time, limit int) ([]*session.Session, error)
	Revoke(ctx context.Context, id uuid.UUID, userID uuid.UUID, now time.Time) error
	RevokeAll(ctx context.Context, userID uuid.UUID, now time.Time) ([]uuid.UUID, error)
	RotateRefreshTokenHash(ctx context.Context, id uuid.UUID, oldHash string, newHash string, clientIP netip.Addr, userAgent string, expiresAt time.Time, now time.Time) (*session.Session, error)
}

type UserRepository interface {
	Availability(ctx context.Context, email *string, username *string) (bool, bool, error)
	Create(ctx context.Context, u *user.User) (*user.User, error)
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*user.User, error)
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	ListDeleteScheduled(ctx context.Context, currentTime time.Time, limitVal int) ([]*user.User, error)
	SetDeleteSchedule(ctx context.Context, id uuid.UUID, deleteScheduledAt time.Time, disabledAt time.Time, updatedAt time.Time) (*user.User, error)
	SetDisabled(ctx context.Context, id uuid.UUID, disabledAt time.Time, updatedAt time.Time) (*user.User, error)
	Update(ctx context.Context, u *user.User) (*user.User, error)
	UpdateBatch(ctx context.Context, users []*user.User) ([]*user.User, error)
	UpdateEmail(ctx context.Context, id uuid.UUID, email string, updatedAt time.Time) (*user.User, error)
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string, updatedAt time.Time) (*user.User, error)
	UpdatePhone(ctx context.Context, id uuid.UUID, phone string, updatedAt time.Time) (*user.User, error)
	UpdatePresence(ctx context.Context, id uuid.UUID, presence presence.Presence, presenceUntil time.Time, updatedAt time.Time) (*user.User, error)
	UpdateProfile(ctx context.Context, id uuid.UUID, displayName string, bio string, avatarURL string, bannerColor string, updatedAt time.Time) (*user.User, error)
	UpdateUsername(ctx context.Context, id uuid.UUID, username string, updatedAt time.Time) (*user.User, error)
	Verify(ctx context.Context, id uuid.UUID, verifiedAt time.Time, updatedAt time.Time) (*user.User, error)
}

type CachedUserRepository interface {
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*user.User, error)
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
