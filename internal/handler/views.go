package handler

import (
	"encoding/json"
	"time"

	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/session"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

type OutboxEventResponse struct {
	ID             uuid.UUID       `json:"id"`
	EventType      string          `json:"event_type"`
	Payload        json.RawMessage `json:"payload"`
	Status         string          `json:"status"`
	Attempts       int32           `json:"attempts"`
	MaxAttempts    int32           `json:"max_attempts"`
	NextAttemptAt  time.Time       `json:"next_attempt_at"`
	ProcessedAt    *time.Time      `json:"processed_at,omitempty"`
	LockedBy       *uuid.UUID      `json:"locked_by,omitempty"`
	LeaseExpiresAt *time.Time      `json:"lease_expires_at,omitempty"`
	LastError      *string         `json:"last_error,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func ToOutboxEventResponse(e outbox.Event) OutboxEventResponse {
	return OutboxEventResponse{
		ID:             e.ID,
		EventType:      e.EventType,
		Payload:        e.Payload,
		Status:         e.GetStatus().String(),
		Attempts:       e.Attempts,
		MaxAttempts:    e.MaxAttempts,
		NextAttemptAt:  e.NextAttemptAt,
		ProcessedAt:    ptr.Map(e.ProcessedAt),
		LockedBy:       ptr.Map(e.LockedBy),
		LeaseExpiresAt: ptr.Map(e.LeaseExpiresAt),
		LastError:      ptr.Map(e.LastError),
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}

// type RelationshipResponse struct {
// 	UserID      uuid.UUID            `json:"user_id"`
// 	PeerID      uuid.UUID            `json:"peer_id"`
// 	Variant     relationship.Variant `json:"type"`
// 	ActorID     uuid.UUID            `json:"actor_id"`
// 	IsInitiator bool                 `json:"is_initiator"`
// 	CreatedAt   time.Time            `json:"created_at"`
// 	UpdatedAt   time.Time            `json:"updated_at"`
// 	Username    string               `json:"username"`
// 	DisplayName string               `json:"display_name"`
// 	AvatarURL   *string              `json:"avatar_url"`
// 	Presence    *presence.Presence   `json:"presence"`
// 	ChannelID   *uuid.UUID           `json:"channel_id"`
// }

// func ToRelationshipResponse(
// 	r relationship.Relationship,
// 	queryingUserID uuid.UUID,
// 	peerUsername string,
// 	peerDisplayName string,
// 	peerAvatarURL *string,
// 	peerPresence *presence.Presence,
// 	channelID *uuid.UUID,
// ) RelationshipResponse {
// 	return RelationshipResponse{
// 		UserID:      queryingUserID,
// 		PeerID:      r.GetPeerID(queryingUserID),
// 		Variant:     r.Variant,
// 		ActorID:     r.ActorID,
// 		IsInitiator: r.ActorID == queryingUserID,
// 		CreatedAt:   r.CreatedAt,
// 		UpdatedAt:   r.UpdatedAt,
// 		Username:    peerUsername,
// 		DisplayName: peerDisplayName,
// 		AvatarURL:   peerAvatarURL,
// 		Presence:    peerPresence,
// 		ChannelID:   channelID,
// 	}
// }

type SessionResponse struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	LastSeenAt time.Time `json:"last_seen_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	ClientIP   string    `json:"client_ip"`
	UserAgent  string    `json:"user_agent"`
	OS         string    `json:"os"`
	Browser    string    `json:"browser"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func ToSessionResponse(s session.Session) SessionResponse {
	return SessionResponse{
		ID:         s.ID(),
		UserID:     s.UserID(),
		LastSeenAt: s.LastSeenAt(),
		ExpiresAt:  s.ExpiresAt(),
		ClientIP:   s.ClientIP().String(),
		UserAgent:  s.UserAgent(),
		OS:         s.OS(),
		Browser:    s.Browser(),
		CreatedAt:  s.CreatedAt(),
		UpdatedAt:  s.UpdatedAt(),
	}
}

type UserResponse struct {
	ID                uuid.UUID          `json:"id"`
	Username          string             `json:"username"`
	PreferredPresence *presence.Presence `json:"preferred_presence,omitempty"`
	IsVerified        bool               `json:"is_verified"`
	CreatedAt         time.Time          `json:"created_at"`
}

func ToUserResponse(u *user.User) UserResponse {
	return UserResponse{
		ID:                u.ID(),
		Username:          u.Username().String(),
		PreferredPresence: u.PreferredPresence(),
		IsVerified:        u.IsVerified(),
		CreatedAt:         u.CreatedAt(),
	}
}

type UserProfileResponse struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func ToUserProfileResponse(userID uuid.UUID, p user.Profile) UserProfileResponse {
	return UserProfileResponse{
		UserID:      userID,
		DisplayName: p.DisplayName().String(),
		AvatarURL:   p.AvatarURL(),
		CreatedAt:   p.CreatedAt(),
		UpdatedAt:   p.UpdatedAt(),
	}
}

type UserMeResponse struct {
	ID                uuid.UUID          `json:"id"`
	Email             string             `json:"email"`
	Username          string             `json:"username"`
	DisplayName       string             `json:"display_name"`
	AvatarURL         *string            `json:"avatar_url,omitempty"`
	PreferredPresence *presence.Presence `json:"preferred_presence,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

func ToUserMeResponse(u user.User) UserMeResponse {
	prof := u.Profile()

	return UserMeResponse{
		ID:                u.ID(),
		Email:             u.Email().String(),
		Username:          u.Username().String(),
		DisplayName:       prof.DisplayName().String(),
		AvatarURL:         prof.AvatarURL(),
		PreferredPresence: u.PreferredPresence(),
		CreatedAt:         u.CreatedAt(),
		UpdatedAt:         u.UpdatedAt(),
	}
}
