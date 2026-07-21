package relationship

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("relationship not found")

type UpsertParams struct {
	User1ID uuid.UUID
	User2ID uuid.UUID
	Type    Type
	ActorID uuid.UUID
}

type GetParams struct {
	User1ID uuid.UUID
	User2ID uuid.UUID
}

type DeleteParams struct {
	User1ID uuid.UUID
	User2ID uuid.UUID
}

type DeleteVerifiedParams struct {
	User1ID uuid.UUID
	User2ID uuid.UUID
	ActorID uuid.UUID
}

type Repository interface {
	ListByUserID(ctx context.Context, userID uuid.UUID, relType Type) ([]Relationship, error)
	Get(ctx context.Context, p GetParams) (Relationship, error)
	GetForUpdate(ctx context.Context, p GetParams) (Relationship, error)
	Upsert(ctx context.Context, p UpsertParams) (Relationship, error)
	Delete(ctx context.Context, p DeleteParams) error
	DeleteVerified(ctx context.Context, p DeleteVerifiedParams) error
}
