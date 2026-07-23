package auth

import (
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/session"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"
	"context"
	"time"

	"github.com/google/uuid"
)

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}

type OutboxRepository interface {
	Publish(ctx context.Context, p outbox.PublishParams) (outbox.Event, error)
}

type SessionRepository interface {
	Create(ctx context.Context, s *session.Session) error
	Save(ctx context.Context, s *session.Session) error
	Get(ctx context.Context, id uuid.UUID) (*session.Session, error)
}

type SessionStore interface {
	Delete(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*session.Session, error)
	Set(ctx context.Context, sess *session.Session) error
}

type ShieldStore interface {
	GetCooldown(ctx context.Context, scope, action, identifier string) (bool, error)
	SetCooldown(ctx context.Context, scope, action, identifier string, ttl time.Duration) error
	IsLocked(ctx context.Context, key string) (bool, error)
	Lockout(ctx context.Context, key string, duration time.Duration) error
	IncrementFailures(ctx context.Context, key string, window time.Duration) (int64, error)
	ResetFailures(ctx context.Context, key string) error
	IsTokenConsumed(ctx context.Context, tokenID string) (bool, error)
	MarkTokenConsumed(ctx context.Context, tokenID string, ttl time.Duration) error
}

type TicketStore interface {
	SetTicket(ctx context.Context, ticketID uuid.UUID, userID uuid.UUID, ttl time.Duration) error
	ConsumeTicket(ctx context.Context, ticketID uuid.UUID) (uuid.UUID, error)
}

type TokenProvider interface {
	GeneratePair(uid, sid uuid.UUID) (token.Pair, error)
	GeneratePasswordReset(userID uuid.UUID) (string, time.Time, error)
	GenerateEmailVerify(userID uuid.UUID) (string, time.Time, error)
	VerifyPasswordReset(tokenStr string) (*token.Claims, error)
	VerifyEmailVerify(tokenStr string) (*token.Claims, error)
	VerifyRefresh(tokenStr string) (*token.Claims, error)
}

type RegisterUserTxParams struct {
	User         *user.User
	Session      *session.Session
	OutboxParams outbox.PublishParams
}

type UserRepository interface {
	CheckAvailability(ctx context.Context, email user.Email, username user.Username) (bool, bool, error)
	Create(ctx context.Context, u *user.User) error
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetByEmail(ctx context.Context, email user.Email) (*user.User, error)
	GetByUsername(ctx context.Context, username user.Username) (*user.User, error)
	Save(ctx context.Context, u *user.User) error
	SaveProfile(ctx context.Context, u *user.User) error
}

type Service struct {
	outbox       OutboxRepository
	sessions     SessionRepository
	users        UserRepository
	sessionStore SessionStore
	shield       ShieldStore
	tickets      TicketStore
	tokens       TokenProvider
	tx           TX
}

func NewService(
	outbox OutboxRepository,
	sessions SessionRepository,
	users UserRepository,
	sessionStore SessionStore,
	shield ShieldStore,
	tickets TicketStore,
	tokens TokenProvider,
	tx TX,
) Service {
	return Service{
		tx:           tx,
		outbox:       outbox,
		sessions:     sessions,
		sessionStore: sessionStore,
		tickets:      tickets,
		tokens:       tokens,
		users:        users,
	}
}
