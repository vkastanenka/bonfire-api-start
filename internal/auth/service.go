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

type OutboxRepository interface {
	Publish(ctx context.Context, p outbox.PublishParams) (outbox.Event, error)
}

type SessionRepository interface {
	Create(ctx context.Context, s *session.Session) error
}

type SessionStore interface {
	Delete(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*session.Session, error)
	Set(ctx context.Context, sess *session.Session) error
}

type ShieldStore interface {
	GetCooldown(ctx context.Context, scope, action, identifier string) (bool, error)
	SetCooldown(ctx context.Context, scope, action, identifier string, ttl time.Duration) error
	IncrementFailures(ctx context.Context, key string, window time.Duration) (int64, error)
	Lockout(ctx context.Context, key string, duration time.Duration) error
	IsLocked(ctx context.Context, key string) (bool, error)
	ResetFailures(ctx context.Context, key string) error
}

type TokenProvider interface {
	GeneratePair(uid, sid uuid.UUID) (token.Pair, error)
	GeneratePasswordReset(userID uuid.UUID) (string, time.Time, error)
}

type UserService interface {
	GetByEmail(ctx context.Context, rawEmail string) (*user.User, error)
}

type Service struct {
	outbox       OutboxRepository
	sessions     SessionRepository
	sessionCache SessionStore
	shield       ShieldStore
	tokens        TokenProvider
	users        UserService
}

// type Service struct {
// 	cooldown cooldown.Repository
// 	user     user.Service
// 	token    *token.Provider
// 	// store       repository.Store
// 	// cache       cache.Store
// 	// session     *session.Service
// 	// user        *user.Service
// 	// flightGroup singleflight.Group
// }

func NewService(
// store repository.Store,
// cache cache.Store,
// token *token.Manager,
// session *session.Service,
// user *user.Service,
) *Service {
	return &Service{
		// store:       store,
		// cache:       cache,
		// token:       token,
		// session:     session,
		// user:        user,
		// flightGroup: singleflight.Group{},
	}
}
