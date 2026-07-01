package relationship

import (
	"bytes"
	"context"
	"time"

	"bonfire-api/internal/presence"
	"bonfire-api/internal/repository"

	"github.com/google/uuid"
)

// --- relationship Type ---

type Type int16

const (
	TypePending Type = 0
	TypeFriends Type = 1
	TypeBlocked Type = 2
)

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

// --- relationship Status ---

type Status string

const (
	StatusAll     Status = "all"
	StatusFriends Status = "friends"
	StatusOnline  Status = "online"
	StatusPending Status = "pending"
	StatusBlocked Status = "blocked"
)

// --- relationship PresenceProvider ---

type PresenceProvider interface {
	GetBulkActivity(ctx context.Context, userIDs []string) (map[string]presence.Activity, error)
}

// --- relationship View ---

type View struct {
	PeerID      uuid.UUID         `json:"peer_id"`
	Username    string            `json:"username"`
	DisplayName string            `json:"display_name"`
	AvatarURL   string            `json:"avatar_url"`
	ActorID     uuid.UUID         `json:"actor_id"`
	Type        string            `json:"type"`
	Activity    presence.Activity `json:"activity"`
	ChannelID   *uuid.UUID        `json:"channel_id,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

func NewView(row repository.RelationshipPerspective, activity presence.Activity) View {
	var channelID *uuid.UUID
	if row.ChannelID.Valid {
		id := uuid.UUID(row.ChannelID.Bytes)
		channelID = &id
	}

	return View{
		PeerID:      uuid.UUID(row.PeerID.Bytes),
		Username:    row.Username,
		DisplayName: row.DisplayName,
		AvatarURL:   row.AvatarUrl.String,
		ActorID:     uuid.UUID(row.ActorID.Bytes),
		Type:        Type(row.Type).String(),
		Activity:    activity,
		ChannelID:   channelID,
		CreatedAt:   row.CreatedAt.Time,
	}
}

// --- relationship orderUUIDs ---

func orderUUIDs(id1, id2 uuid.UUID) (uuid.UUID, uuid.UUID) {
	if bytes.Compare(id1[:], id2[:]) < 0 {
		return id1, id2
	}
	return id2, id1
}
