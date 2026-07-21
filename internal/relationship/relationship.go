package relationship

import (
	"bytes"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Type int16

const (
	TypeUnknown Type = iota // 0 (Implicitly the Go zero-value)
	TypePending             // 1
	TypeFriends             // 2
	TypeBlocked             // 3
	typeMax                 // 4 (Boundary marker)
)

func (t Type) Valid() bool {
	return t > TypeUnknown && t < typeMax
}

func (t Type) String() string {
	switch t {
	case TypePending:
		return "pending"
	case TypeFriends:
		return "friends"
	case TypeBlocked:
		return "blocked"
	default:
		return "unknown"
	}
}

func Parse(s string) Type {
	switch s {
	case "pending":
		return TypePending
	case "friends":
		return TypeFriends
	case "blocked":
		return TypeBlocked
	default:
		return TypeUnknown
	}
}

func (t Type) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", t.String())), nil
}

func (t *Type) UnmarshalJSON(data []byte) error {
	data = bytes.Trim(data, "\"")
	*t = Parse(string(data))
	return nil
}

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
