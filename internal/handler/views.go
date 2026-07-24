package handler

import (
	"encoding/json"
	"time"

	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/relationship"
	"bonfire-api/internal/session"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

type OutboxEventResponse struct {
	ID             uuid.UUID       `json:"id"`
	EventType      string          `json:"eventType"`
	Payload        json.RawMessage `json:"payload"`
	Status         string          `json:"status"`
	Attempts       int32           `json:"attempts"`
	MaxAttempts    int32           `json:"maxAttempts"`
	NextAttemptAt  time.Time       `json:"nextAttemptAt"`
	ProcessedAt    *time.Time      `json:"processedAt,omitempty"`
	LockedBy       *uuid.UUID      `json:"lockedBy,omitempty"`
	LeaseExpiresAt *time.Time      `json:"leaseExpiresAt,omitempty"`
	LastError      *string         `json:"lastError,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
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
	UserID            uuid.UUID            `json:"userId"`
	PeerID            uuid.UUID            `json:"peerId"`
	Variant           relationship.Variant `json:"type"`
	ActorID           uuid.UUID            `json:"actorId"`
	IsInitiator       bool                 `json:"isInitiator"`
	CreatedAt         time.Time            `json:"createdAt"`
	UpdatedAt         time.Time            `json:"updatedAt"`
	Username          string               `json:"username"`
	DisplayName       string               `json:"displayName"`
	AvatarURL         *string              `json:"avatarUrl,omitempty"`
	PreferredPresence *presence.Presence   `json:"preferredPresence,omitempty"`
	ChannelID         *uuid.UUID           `json:"channelId,omitempty"`
}

func ToRelationshipResponse(p relationship.Perspective, realTimePresence *presence.Presence) RelationshipResponse {
	var displayName string
	if dn := p.DisplayName(); dn != nil {
		displayName = dn.String()
	}

	return RelationshipResponse{
		UserID:            p.UserID(),
		PeerID:            p.PeerID(),
		Variant:           p.Variant(),
		ActorID:           p.ActorID(),
		IsInitiator:       p.IsInitiator(),
		CreatedAt:         p.CreatedAt(),
		UpdatedAt:         p.UpdatedAt(),
		Username:          p.Username().String(),
		DisplayName:       displayName,
		AvatarURL:         p.AvatarURL(),
		PreferredPresence: realTimePresence,
		ChannelID:         p.ChannelID(),
	}
}

type SessionResponse struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"userId"`
	LastSeenAt time.Time `json:"lastSeenAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
	ClientIP   string    `json:"clientIp"`
	UserAgent  string    `json:"userAgent"`
	OS         string    `json:"os"`
	Browser    string    `json:"browser"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
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
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	AvatarURL   *string   `json:"avatarUrl,omitempty"`
	IsVerified  bool      `json:"isVerified"`
	CreatedAt   time.Time `json:"createdAt"`
}

func ToUserResponse(u user.User) UserResponse {
	prof := u.Profile()

	return UserResponse{
		ID:          u.ID(),
		Username:    u.Username().String(),
		DisplayName: prof.DisplayName().String(),
		AvatarURL:   prof.AvatarURL(),
		IsVerified:  u.VerifiedAt() != nil,
		CreatedAt:   u.CreatedAt(),
	}
}

type UserMeResponse struct {
	ID                uuid.UUID          `json:"id"`
	Email             string             `json:"email"`
	Username          string             `json:"username"`
	DisplayName       string             `json:"displayName"`
	AvatarURL         *string            `json:"avatarUrl,omitempty"`
	PreferredPresence *presence.Presence `json:"preferredPresence,omitempty"`
	IsVerified        bool               `json:"isVerified"`
	VerifiedAt        *time.Time         `json:"verifiedAt,omitempty"`
	CreatedAt         time.Time          `json:"createdAt"`
	UpdatedAt         time.Time          `json:"updatedAt"`
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
		IsVerified:        u.VerifiedAt() != nil,
		VerifiedAt:        u.VerifiedAt(),
		CreatedAt:         u.CreatedAt(),
		UpdatedAt:         u.UpdatedAt(),
	}
}
