package auth

import (
	"bonfire-api/internal/session"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"
	"context"

	"github.com/google/uuid"
)

type SessionCache interface {
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteBatch(ctx context.Context, ids []uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*session.Session, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*session.Session, []uuid.UUID, error)
	Set(ctx context.Context, sess *session.Session) error
	SetBatch(ctx context.Context, sessions map[uuid.UUID]*session.Session) error
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

type WSTicketCache interface {
	Print(ctx context.Context, ticketID uuid.UUID, userID uuid.UUID, sessionID uuid.UUID) error
	Punch(ctx context.Context, ticketID uuid.UUID) (uuid.UUID, uuid.UUID, error)
}
