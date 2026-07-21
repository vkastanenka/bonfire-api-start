package relationship

import (
	"bonfire-api/internal/user"
	"time"

	"github.com/google/uuid"
)

type PerspectiveView struct {
	UserID      uuid.UUID      `json:"user_id"`
	PeerID      uuid.UUID      `json:"peer_id"`
	Type        Type           `json:"type"`
	ActorID     uuid.UUID      `json:"actor_id"`
	IsInitiator bool           `json:"is_initiator"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Username    string         `json:"username"`
	DisplayName string         `json:"display_name"`
	AvatarURL   *string        `json:"avatar_url"`
	Presence    *user.Presence `json:"presence"`
	ChannelID   *uuid.UUID     `json:"channel_id"`
}

func ToPerspectiveView(
	r Relationship,
	queryingUserID uuid.UUID,
	peerUsername string,
	peerDisplayName string,
	peerAvatarURL *string,
	peerPresence *user.Presence,
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
