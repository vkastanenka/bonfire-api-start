package relationship

import (
	"bytes"
	"fmt"
	"time"

	"bonfire-api/internal/presence"
	"bonfire-api/internal/repository"

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

func FromRepository(row repository.Relationship) Relationship {
	return Relationship{
		User1ID:   uuid.UUID(row.User1ID.Bytes),
		User2ID:   uuid.UUID(row.User2ID.Bytes),
		ActorID:   uuid.UUID(row.ActorID.Bytes),
		Type:      Type(row.Type),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}

type PerspectiveView struct {
	UserID      uuid.UUID          `json:"user_id"`
	PeerID      uuid.UUID          `json:"peer_id"`
	Type        Type               `json:"type"`
	ActorID     uuid.UUID          `json:"actor_id"`
	IsInitiator bool               `json:"is_initiator"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	Username    string             `json:"username"`
	DisplayName string             `json:"display_name"`
	AvatarURL   *string            `json:"avatar_url"`
	Presence    *presence.Presence `json:"presence"`
	ChannelID   *uuid.UUID         `json:"channel_id"`
}

func ToPerspectiveView(
	r Relationship,
	queryingUserID uuid.UUID,
	peerUsername string,
	peerDisplayName string,
	peerAvatarURL *string,
	peerPresence *presence.Presence,
	channelID *uuid.UUID,
) PerspectiveView {
	return PerspectiveView{
		UserID:      queryingUserID,
		PeerID:      r.GetPeerID(queryingUserID),
		Type:        r.Type,
		ActorID:     r.ActorID,
		IsInitiator: r.ActorID == queryingUserID,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
		Username:    peerUsername,
		DisplayName: peerDisplayName,
		AvatarURL:   peerAvatarURL,
		Presence:    peerPresence,
		ChannelID:   channelID,
	}
}
