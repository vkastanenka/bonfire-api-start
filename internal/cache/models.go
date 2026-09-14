package cache

import (
	"bonfire-api/internal/channel"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/session"
	"bonfire-api/internal/user"
	"encoding/json"
	"net/netip"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type Channel struct {
	ID            uuid.UUID  `json:"id"`
	Type          int        `json:"type"`
	Name          *string    `json:"name,omitempty"`
	IconURL       *string    `json:"icon_url,omitempty"`
	LastMessageID *uuid.UUID `json:"last_message_id,omitempty"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func ParseChannel(ch *channel.Channel) Channel {
	if ch == nil {
		return Channel{}
	}

	return Channel{
		ID:            ch.ID,
		Type:          int(ch.Type),
		Name:          ch.Name,
		IconURL:       ch.IconURL,
		LastMessageID: ch.LastMessageID,
		LastMessageAt: ch.LastMessageAt,
		CreatedAt:     ch.CreatedAt,
		UpdatedAt:     ch.UpdatedAt,
	}
}

func (c Channel) ToDomain() *channel.Channel {
	return channel.ReconstituteChannel(
		c.ID,
		channel.ChannelType(c.Type),
		c.Name,
		c.IconURL,
		c.LastMessageID,
		c.LastMessageAt,
		c.CreatedAt,
		c.UpdatedAt,
	)
}

type Member struct {
	ChannelID         uuid.UUID  `json:"channel_id"`
	UserID            uuid.UUID  `json:"user_id"`
	LastReadMessageID *uuid.UUID `json:"last_read_message_id,omitempty"`
	LastReadMessageAt *time.Time `json:"last_read_message_at,omitempty"`
	PinnedAt          *time.Time `json:"pinned_at,omitempty"`
	MutedUntil        *time.Time `json:"muted_until,omitempty"`
	MentionCount      int        `json:"mention_count"`
	IsVisible         bool       `json:"is_visible"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func ParseMember(m *channel.Member) Member {
	if m == nil {
		return Member{}
	}

	return Member{
		ChannelID:         m.ChannelID,
		UserID:            m.UserID,
		LastReadMessageID: m.LastReadMessageID,
		LastReadMessageAt: m.LastReadMessageAt,
		PinnedAt:          m.PinnedAt,
		MutedUntil:        m.MutedUntil,
		MentionCount:      m.MentionCount,
		IsVisible:         m.IsVisible,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}

func (m Member) ToDomain() *channel.Member {
	return channel.ReconstituteMember(
		m.ChannelID,
		m.UserID,
		m.LastReadMessageID,
		m.LastReadMessageAt,
		m.PinnedAt,
		m.MutedUntil,
		m.MentionCount,
		m.IsVisible,
		m.CreatedAt,
		m.UpdatedAt,
	)
}

type Message struct {
	ID               uuid.UUID       `json:"id"`
	ChannelID        uuid.UUID       `json:"channel_id"`
	AuthorID         *uuid.UUID      `json:"author_id,omitempty"`
	Type             int             `json:"type"`
	Content          *string         `json:"content,omitempty"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
	ReplyToMessageID *uuid.UUID      `json:"reply_to_message_id,omitempty"`
	ForwardMessageID *uuid.UUID      `json:"forward_message_id,omitempty"`
	ForwardChannelID *uuid.UUID      `json:"forward_channel_id,omitempty"`
	PinnedAt         *time.Time      `json:"pinned_at,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	EditedAt         *time.Time      `json:"edited_at,omitempty"`
}

func ParseMessage(msg *channel.Message) Message {
	if msg == nil {
		return Message{}
	}

	return Message{
		ID:               msg.ID,
		ChannelID:        msg.ChannelID,
		AuthorID:         msg.AuthorID,
		Type:             int(msg.Type),
		Content:          msg.Content,
		Metadata:         msg.Metadata,
		ReplyToMessageID: msg.ReplyToMessageID,
		ForwardMessageID: msg.ForwardMessageID,
		ForwardChannelID: msg.ForwardChannelID,
		PinnedAt:         msg.PinnedAt,
		CreatedAt:        msg.CreatedAt,
		UpdatedAt:        msg.UpdatedAt,
		EditedAt:         msg.EditedAt,
	}
}

func (m Message) ToDomain() *channel.Message {
	return channel.ReconstituteMessage(
		m.ID,
		m.ChannelID,
		m.AuthorID,
		channel.MessageType(m.Type),
		m.Content,
		m.Metadata,
		m.ReplyToMessageID,
		m.ForwardMessageID,
		m.ForwardChannelID,
		m.PinnedAt,
		m.CreatedAt,
		m.UpdatedAt,
		m.EditedAt,
	)
}

type Session struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	RefreshTokenHash string     `json:"refresh_token_hash"`
	ClientIP         netip.Addr `json:"client_ip"`
	UserAgent        string     `json:"user_agent"`
	OS               string     `json:"os"`
	Client           string     `json:"client"`
	ExpiresAt        time.Time  `json:"expires_at"`
	LastSeenAt       time.Time  `json:"last_seen_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func ParseSession(s *session.Session) Session {
	if s == nil {
		return Session{}
	}

	return Session{
		ID:               s.ID,
		UserID:           s.UserID,
		RefreshTokenHash: s.RefreshTokenHash,
		ClientIP:         s.ClientIP,
		UserAgent:        s.UserAgent,
		OS:               s.OS,
		Client:           s.Client,
		ExpiresAt:        s.ExpiresAt,
		LastSeenAt:       s.LastSeenAt,
		RevokedAt:        s.RevokedAt,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

func (s Session) ToDomain() *session.Session {
	return session.Reconstitute(
		s.ID,
		s.UserID,
		s.RefreshTokenHash,
		s.ClientIP,
		s.UserAgent,
		s.OS,
		s.Client,
		s.ExpiresAt,
		s.LastSeenAt,
		s.RevokedAt,
		s.CreatedAt,
		s.UpdatedAt,
	)
}

type User struct {
	ID                     uuid.UUID          `json:"id"`
	Email                  string             `json:"email"`
	Username               string             `json:"username"`
	DisplayName            string             `json:"display_name"`
	PasswordHash           string             `json:"password_hash"`
	Phone                  *string            `json:"phone,omitempty"`
	Bio                    *string            `json:"bio,omitempty"`
	AvatarURL              *string            `json:"avatar_url,omitempty"`
	BannerColor            *string            `json:"banner_color,omitempty"`
	PreferredPresence      *presence.Presence `json:"preferred_presence,omitempty"`
	PreferredPresenceUntil *time.Time         `json:"preferred_presence_until,omitempty"`
	VerifiedAt             *time.Time         `json:"verified_at,omitempty"`
	DisabledAt             *time.Time         `json:"disabled_at,omitempty"`
	DeleteScheduledAt      *time.Time         `json:"delete_scheduled_at,omitempty"`
	CreatedAt              time.Time          `json:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at"`
}

func ParseUser(u *user.User) User {
	if u == nil {
		return User{}
	}

	return User{
		ID:                     u.ID,
		Email:                  u.Email,
		Username:               u.Username,
		DisplayName:            u.DisplayName,
		PasswordHash:           u.PasswordHash,
		Phone:                  u.Phone,
		Bio:                    u.Bio,
		AvatarURL:              u.AvatarURL,
		BannerColor:            u.BannerColor,
		PreferredPresence:      u.PreferredPresence,
		PreferredPresenceUntil: u.PreferredPresenceUntil,
		VerifiedAt:             u.VerifiedAt,
		DisabledAt:             u.DisabledAt,
		DeleteScheduledAt:      u.DeleteScheduledAt,
		CreatedAt:              u.CreatedAt,
		UpdatedAt:              u.UpdatedAt,
	}
}

func (u User) ToDomain() *user.User {
	return user.Reconstitute(
		u.ID,
		u.Email,
		u.Username,
		u.PasswordHash,
		u.Phone,
		u.DisplayName,
		u.Bio,
		u.AvatarURL,
		u.BannerColor,
		u.PreferredPresence,
		u.PreferredPresenceUntil,
		u.VerifiedAt,
		u.DisabledAt,
		u.DeleteScheduledAt,
		u.CreatedAt,
		u.UpdatedAt,
	)
}

func marshalUser(usr *user.User) ([]byte, error) {
	if usr == nil {
		return nil, nil
	}

	dto := ParseUser(usr)
	bytes, err := json.Marshal(dto)
	if err != nil {
		return nil, errs.Internal("Failed to marshal user json.").
			Meta("scope", redis.ScopeUser.String()).
			Wrap(err)
	}
	return bytes, nil
}

func unmarshalUser(data []byte) (*user.User, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var dto User
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}
	return dto.ToDomain(), nil
}

func marshalSession(sess *session.Session) ([]byte, error) {
	if sess == nil {
		return nil, nil
	}

	dto := ParseSession(sess)
	bytes, err := json.Marshal(dto)
	if err != nil {
		return nil, errs.Internal("Failed to marshal session json.").
			Meta("scope", redis.ScopeSession.String()).
			Wrap(err)
	}
	return bytes, nil
}

func unmarshalSession(data []byte) (*session.Session, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var dto Session
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}
	return dto.ToDomain(), nil
}

func marshalChannel(ch *channel.Channel) ([]byte, error) {
	if ch == nil {
		return nil, nil
	}

	dto := ParseChannel(ch)
	bytes, err := json.Marshal(dto)
	if err != nil {
		return nil, errs.Internal("Failed to marshal channel json.").
			Meta("scope", redis.ScopeChannel.String()).
			Wrap(err)
	}
	return bytes, nil
}

func unmarshalChannel(data []byte) (*channel.Channel, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var dto Channel
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}
	return dto.ToDomain(), nil
}

func unmarshalMessage(data []byte) (*channel.Message, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var dto Message
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}
	return dto.ToDomain(), nil
}

func ParsePresence(val string) presence.Presence {
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return presence.PresenceOffline
	}

	p, err := presence.Parse(parsed)
	if err != nil {
		return presence.PresenceOffline
	}

	return p
}
