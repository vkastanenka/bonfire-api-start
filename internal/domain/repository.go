package domain

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type TxUnit interface {
	CreateUser(ctx context.Context, email, username, passwordHash string) (pgtype.UUID, error)
	CreateUserProfile(ctx context.Context, userID pgtype.UUID, displayName *string) (pgtype.UUID, error)
}

type DBRepository interface {
	WithTx(ctx context.Context, fn func(tx TxUnit) error) error
}
