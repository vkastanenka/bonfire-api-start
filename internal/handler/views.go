package handler

import (
	"encoding/json"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/relationship"
	"bonfire-api/internal/session"

	"github.com/google/uuid"
)

type OutboxEventResponse struct {
	ID             outbox.EventID  `json:"id"`
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
		ID:             e.ID(),
		EventType:      e.EventType(),
		Payload:        e.Payload(),
		Status:         e.Status().String(),
		Attempts:       e.Attempts(),
		MaxAttempts:    e.MaxAttempts(),
		NextAttemptAt:  e.NextAttemptAt(),
		ProcessedAt:    ptr.Map(e.ProcessedAt()),
		LockedBy:       ptr.Map(e.LockedBy()),
		LeaseExpiresAt: ptr.Map(e.LeaseExpiresAt()),
		LastError:      ptr.Map(e.LastError()),
		CreatedAt:      e.CreatedAt(),
		UpdatedAt:      e.UpdatedAt(),
	}
}

type RelationshipResponse struct {
	PeerID      uuid.UUID            `json:"peerId"`
	Variant     relationship.Variant `json:"type"`
	IsInitiator bool                 `json:"isInitiator"`
	CreatedAt   time.Time            `json:"createdAt"`
	UpdatedAt   time.Time            `json:"updatedAt"`
	Username    string               `json:"username"`
	DisplayName string               `json:"displayName"`
	AvatarURL   *string              `json:"avatarUrl,omitempty"`
	Presence    *presence.Presence   `json:"presence,omitempty"`
	ChannelID   *uuid.UUID           `json:"channelId,omitempty"`
}

func ToRelationshipResponse(p relationship.Perspective, realTimePresence *presence.Presence) RelationshipResponse {
	displayName := p.Username().String()
	if dn := p.DisplayName(); dn != nil && dn.String() != "" {
		displayName = dn.String()
	}

	var safePresence *presence.Presence
	if p.Variant() != relationship.VariantBlocked {
		safePresence = realTimePresence
	}

	return RelationshipResponse{
		PeerID:      p.PeerID(),
		Variant:     p.Variant(),
		IsInitiator: p.IsInitiator(),
		CreatedAt:   p.CreatedAt(),
		UpdatedAt:   p.UpdatedAt(),
		Username:    p.Username().String(),
		DisplayName: displayName,
		AvatarURL:   p.AvatarURL(),
		Presence:    safePresence,
		ChannelID:   p.ChannelID(),
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

type ChannelResponse struct {
	ID        uuid.UUID    `json:"id"`
	Type      channel.Type `json:"type"`
	OwnerID   *uuid.UUID   `json:"ownerId,omitempty"`
	Name      *string      `json:"name,omitempty"`
	IconURL   *string      `json:"iconUrl,omitempty"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
}

func ToChannelResponse(c channel.Channel) ChannelResponse {
	return ChannelResponse{
		ID:        c.ID(),
		Type:      c.Type(),
		OwnerID:   c.OwnerID(),
		Name:      c.Name(), // c.Name() already returns *string
		IconURL:   c.IconURL(),
		CreatedAt: c.CreatedAt(),
		UpdatedAt: c.UpdatedAt(),
	}
}

type MessageResponse struct {
	ID        uuid.UUID  `json:"id"`
	ChannelID uuid.UUID  `json:"channelId"`
	AuthorID  *uuid.UUID `json:"authorId,omitempty"`
	ReplyToID *uuid.UUID `json:"replyToId,omitempty"`
	Content   string     `json:"content"`
	Pinned    bool       `json:"pinned"`
	CreatedAt time.Time  `json:"createdAt"`
	EditedAt  *time.Time `json:"editedAt,omitempty"`
}

func ToMessageResponse(m channel.Message) MessageResponse {
	return MessageResponse{
		ID:        m.ID(),
		ChannelID: m.ChannelID(),
		AuthorID:  m.AuthorID(),
		ReplyToID: m.ReplyToMessageID(),
		Content:   m.Content(),  // m.Content() returns string directly
		Pinned:    m.IsPinned(), // getter method is IsPinned()
		CreatedAt: m.CreatedAt(),
		EditedAt:  m.EditedAt(), // getter method is EditedAt()
	}
}
