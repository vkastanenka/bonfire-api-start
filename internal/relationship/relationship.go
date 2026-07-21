package relationship

import (
	"time"

	"github.com/google/uuid"
)

type Relationship struct {
	User1ID   uuid.UUID `json:"user_1_id"`
	User2ID   uuid.UUID `json:"user_2_id"`
	ActorID   uuid.UUID `json:"actor_id"`
	Type      Type      `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r Relationship) IsPending() bool {
	return r.Type == TypePending
}

func (r Relationship) IsFriends() bool {
	return r.Type == TypeFriends
}

func (r Relationship) IsBlocked() bool {
	return r.Type == TypeBlocked
}

func (r Relationship) GetPeerID(userID uuid.UUID) uuid.UUID {
	if userID == r.User1ID {
		return r.User2ID
	}
	return r.User1ID
}
