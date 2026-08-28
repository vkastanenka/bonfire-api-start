package auth

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/session"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"
	"context"
	"time"
)

type UserRepository interface {
	Availability(ctx context.Context, email *user.Email, username *user.Username) (bool, bool, error)
	Create(ctx context.Context, u *user.User) (*user.User, error)
	Get(ctx context.Context, id fields.ID) (*user.User, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*user.User, error)
	GetByEmail(ctx context.Context, email user.Email) (*user.User, error)
	ListDeleteScheduled(ctx context.Context, currentTime fields.Timestamp, limitVal int) ([]*user.User, error)
	SetDeleteSchedule(ctx context.Context, id fields.ID, deleteScheduledAt fields.Timestamp, disabledAt fields.Timestamp, updatedAt fields.Timestamp) (*user.User, error)
	SetDisabled(ctx context.Context, id fields.ID, disabledAt fields.Timestamp, updatedAt fields.Timestamp) (*user.User, error)
	Update(ctx context.Context, u *user.User) (*user.User, error)
	UpdateBatch(ctx context.Context, users []*user.User) ([]*user.User, error)
	UpdateEmail(ctx context.Context, id fields.ID, email user.Email, updatedAt fields.Timestamp) (*user.User, error)
	UpdatePasswordHash(ctx context.Context, id fields.ID, passwordHash user.PasswordHash, updatedAt fields.Timestamp) (*user.User, error)
	UpdatePhone(ctx context.Context, id fields.ID, phone user.Phone, updatedAt fields.Timestamp) (*user.User, error)
	UpdatePresence(ctx context.Context, id fields.ID, presence user.PreferredPresence, presenceUntil fields.Timestamp, updatedAt fields.Timestamp) (*user.User, error)
	UpdateProfile(ctx context.Context, id fields.ID, displayName user.DisplayName, bio user.Bio, avatarURL fields.URL, bannerColor fields.HexColor, updatedAt fields.Timestamp) (*user.User, error)
	UpdateUsername(ctx context.Context, id fields.ID, username user.Username, updatedAt fields.Timestamp) (*user.User, error)
	Verify(ctx context.Context, id fields.ID, verifiedAt fields.Timestamp, updatedAt fields.Timestamp) (*user.User, error)
}

type SessionRepository interface {
	Create(ctx context.Context, s *session.Session) (*session.Session, error)
	DeleteBatchExpired(ctx context.Context, now time.Time, limitVal int) error
	Get(ctx context.Context, id fields.ID) (*session.Session, error)
	ListValidByUserID(ctx context.Context, userID fields.ID, now fields.Timestamp, limit int) ([]*session.Session, error)
	Revoke(ctx context.Context, id fields.ID, userID fields.ID, now fields.Timestamp) error
	RevokeAll(ctx context.Context, userID fields.ID, now fields.Timestamp) error
	RotateRefreshTokenHash(ctx context.Context, id fields.ID, oldHash fields.TokenHash, newHash fields.TokenHash, clientIP fields.IP, userAgent fields.UserAgent, expiresAt fields.Timestamp, now fields.Timestamp) (*session.Session, error)
}

type OutboxRepository interface {
	Publish(ctx context.Context, variant string, payload any) error
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}

type TicketCache interface {
	Print(ctx context.Context, ticketID fields.ID, userID fields.ID, sessionID fields.ID) error
	Punch(ctx context.Context, ticketID fields.ID) (fields.ID, fields.ID, error)
}

type TokenProvider interface {
	GenerateAccess(uid fields.ID, sid fields.ID) (string, time.Time, error)
	GenerateEmailVerify(userID fields.ID) (string, time.Time, error)
	GeneratePair(uid fields.ID, sid fields.ID) (token.Pair, error)
	GeneratePasswordReset(userID fields.ID) (string, time.Time, error)
	GenerateRefresh(uid fields.ID, sid fields.ID) (string, time.Time, error)
	VerifyAccess(tokenStr string) (*token.Claims, error)
	VerifyEmailVerify(tokenStr string) (*token.Claims, error)
	VerifyPasswordReset(tokenStr string) (*token.Claims, error)
	VerifyRefresh(tokenStr string) (*token.Claims, error)
}
