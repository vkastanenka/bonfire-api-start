package relationship

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Get(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID) (*Relationship, error)
	GetForUpdate(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID) (*Relationship, error)
	Upsert(ctx context.Context, rel *Relationship) error
	Delete(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID) error
	DeleteVerified(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID, actorID uuid.UUID) error
	GetPerspective(ctx context.Context, userID uuid.UUID, peerID uuid.UUID) (*Perspective, error)
	ListPerspectives(ctx context.Context, userID uuid.UUID, filterVariant *Variant) ([]Perspective, error)
}
