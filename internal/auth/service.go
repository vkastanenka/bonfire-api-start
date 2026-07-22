package auth

import (
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/user"
	"context"
	"time"

	"github.com/google/uuid"
)

type CooldownRepository interface {
	Get(ctx context.Context, scope, action, identifier string) (bool, error)
	Set(ctx context.Context, scope, action, identifier string, ttl time.Duration) error
}

type OutboxRepository interface {
	Publish(ctx context.Context, p outbox.PublishParams) (outbox.Event, error)
}

type TokenProvider interface {
	GeneratePasswordReset(userID uuid.UUID) (string, time.Time, error)
}

type UserService interface {
	GetByEmail(ctx context.Context, rawEmail string) (*user.User, error)
}

type Service struct {
	cooldown CooldownRepository
	outbox   OutboxRepository
	token    TokenProvider
	user     UserService
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
