package handler

import (
	"encoding/json"
	"time"

	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/relationship"
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

type RelationshipResponse struct {
	UserID      uuid.UUID         `json:"user_id"`
	PeerID      uuid.UUID         `json:"peer_id"`
	Type        relationship.Type `json:"type"`
	ActorID     uuid.UUID         `json:"actor_id"`
	IsInitiator bool              `json:"is_initiator"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Username    string            `json:"username"`
	DisplayName string            `json:"display_name"`
	AvatarURL   *string           `json:"avatar_url"`
	Presence    *user.Presence    `json:"presence"`
	ChannelID   *uuid.UUID        `json:"channel_id"`
}

func ToRelationshipResponse(
	r relationship.Relationship,
	queryingUserID uuid.UUID,
	peerUsername string,
	peerDisplayName string,
	peerAvatarURL *string,
	peerPresence *user.Presence,
	channelID *uuid.UUID,
) RelationshipResponse {
	return RelationshipResponse{
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
		ID:         s.ID,
		UserID:     s.UserID,
		LastSeenAt: s.LastSeenAt,
		ExpiresAt:  s.ExpiresAt,
		ClientIP:   s.ClientIP.String(),
		UserAgent:  s.UserAgent,
		OS:         s.OS,
		Browser:    s.Browser,
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
	}
}

type UserResponse struct {
	ID                uuid.UUID     `json:"id"`
	Username          string        `json:"username"`
	PreferredPresence user.Presence `json:"preferred_presence,omitempty"`
	IsVerified        bool          `json:"is_verified"`
	CreatedAt         time.Time     `json:"created_at"`
}

func ToUserResponse(u user.User) UserResponse {
	return UserResponse{
		ID:                u.ID,
		Username:          u.Username.String(), // Extracted string value
		PreferredPresence: u.PreferredPresence,
		IsVerified:        u.IsVerified(),
		CreatedAt:         u.CreatedAt,
	}
}

type UserProfileResponse struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func ToUserProfileResponse(up user.Profile) UserProfileResponse {
	return UserProfileResponse{
		UserID:      up.UserID,
		DisplayName: up.DisplayName.String(), // Extracted string value
		AvatarURL:   ptr.Map(up.AvatarURL),
		CreatedAt:   up.CreatedAt,
		UpdatedAt:   up.UpdatedAt,
	}
}

type UserMeResponse struct {
	ID                uuid.UUID      `json:"id"`
	Email             string         `json:"email"`
	Username          string         `json:"username"`
	DisplayName       string         `json:"display_name"`
	AvatarURL         *string        `json:"avatar_url,omitempty"`
	PreferredPresence *user.Presence `json:"presence,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

func ToUserMeResponse(u user.User, p user.Profile) UserMeResponse {
	return UserMeResponse{
		ID:                u.ID,
		Email:             u.Email.String(),       // Extracted string value
		Username:          u.Username.String(),    // Extracted string value
		DisplayName:       p.DisplayName.String(), // Extracted string value
		AvatarURL:         ptr.Map(p.AvatarURL),
		PreferredPresence: &u.PreferredPresence,
		CreatedAt:         u.CreatedAt,
		UpdatedAt:         u.UpdatedAt,
	}
}
