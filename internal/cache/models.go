package cache

import (
	"bonfire-api/internal/channel"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/session"
	"bonfire-api/internal/user"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type Channel struct {
	ID            uuid.UUID `json:"id"`
	Type          int       `json:"type"`
	Name          string    `json:"name"`
	IconURL       string    `json:"icon_url"`
	LastMessageID uuid.UUID `json:"last_message_id"`
	LastMessageAt time.Time `json:"last_message_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (c Channel) ToDomain() (*channel.Channel, error) {
	id, err := fields.ParseRequiredID("id", c.ID)
	if err != nil {
		return nil, err
	}

	chType, err := channel.ParseChannelType(c.Type)
	if err != nil {
		return nil, err
	}

	name, err := channel.ParseChannelName(c.Name)
	if err != nil {
		return nil, err
	}

	iconURL, err := fields.ParseURL("icon_url", c.IconURL)
	if err != nil {
		return nil, err
	}

	lastMessageID, err := fields.ParseID(c.LastMessageID)
	if err != nil {
		return nil, err
	}

	return channel.ReconstituteChannel(
		id,
		chType,
		name,
		iconURL,
		lastMessageID,
		fields.NewTimestamp(c.LastMessageAt),
		fields.NewTimestamp(c.CreatedAt),
		fields.NewTimestamp(c.UpdatedAt),
	), nil
}

func ParseChannel(ch *channel.Channel) Channel {
	return Channel{
		ID:            ch.ID().UUID(),
		Type:          ch.Type().Int(),
		Name:          ch.Name().String(),
		IconURL:       ch.IconURL().String(),
		LastMessageID: ch.LastMessageID().UUID(),
		LastMessageAt: ch.LastMessageAt().Time(),
		CreatedAt:     ch.CreatedAt().Time(),
		UpdatedAt:     ch.UpdatedAt().Time(),
	}
}

type Member struct {
	ChannelID         uuid.UUID `json:"channel_id"`
	UserID            uuid.UUID `json:"user_id"`
	LastReadMessageID uuid.UUID `json:"last_read_message_id"`
	LastReadMessageAt time.Time `json:"last_read_message_at"`
	PinnedAt          time.Time `json:"pinned_at"`
	MutedUntil        time.Time `json:"muted_until"`
	MentionCount      int       `json:"mention_count"`
	IsVisible         bool      `json:"is_visible"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (m Member) ToDomain() (*channel.Member, error) {
	channelID, err := fields.ParseRequiredID("channel_id", m.ChannelID)
	if err != nil {
		return nil, err
	}

	userID, err := fields.ParseRequiredID("user_id", m.UserID)
	if err != nil {
		return nil, err
	}

	lastReadMessageID, err := fields.ParseID(m.LastReadMessageID)
	if err != nil {
		return nil, err
	}

	return channel.ReconstituteMember(
		channelID,
		userID,
		lastReadMessageID,
		fields.NewTimestamp(m.LastReadMessageAt),
		fields.NewTimestamp(m.PinnedAt),
		fields.NewTimestamp(m.MutedUntil),
		m.MentionCount,
		m.IsVisible,
		fields.NewTimestamp(m.CreatedAt),
		fields.NewTimestamp(m.UpdatedAt),
	), nil
}

func ParseMember(m *channel.Member) Member {
	return Member{
		ChannelID:         m.ChannelID().UUID(),
		UserID:            m.UserID().UUID(),
		LastReadMessageID: m.LastReadMessageID().UUID(),
		LastReadMessageAt: m.LastReadMessageAt().Time(),
		PinnedAt:          m.PinnedAt().Time(),
		MutedUntil:        m.MutedUntil().Time(),
		MentionCount:      m.MentionCount(),
		IsVisible:         m.IsVisible(),
		CreatedAt:         m.CreatedAt().Time(),
		UpdatedAt:         m.UpdatedAt().Time(),
	}
}

type Message struct {
	ID               uuid.UUID       `json:"id"`
	ChannelID        uuid.UUID       `json:"channel_id"`
	AuthorID         uuid.UUID       `json:"author_id"`
	Type             int             `json:"type"`
	Content          json.RawMessage `json:"content"`
	Metadata         json.RawMessage `json:"metadata"`
	ReplyToMessageID uuid.UUID       `json:"reply_to_message_id"`
	ForwardMessageID uuid.UUID       `json:"forward_message_id"`
	ForwardChannelID uuid.UUID       `json:"forward_channel_id"`
	PinnedAt         time.Time       `json:"pinned_at"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	EditedAt         time.Time       `json:"edited_at"`
}

func (m Message) ToDomain() (*channel.Message, error) {
	id, err := fields.ParseRequiredID("id", m.ID)
	if err != nil {
		return nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", m.ChannelID)
	if err != nil {
		return nil, err
	}

	authorID, err := fields.ParseRequiredID("author_id", m.AuthorID)
	if err != nil {
		return nil, err
	}

	replyToMessageID, err := fields.ParseID(m.ReplyToMessageID)
	if err != nil {
		return nil, err
	}

	forwardedMessageID, err := fields.ParseID(m.ForwardMessageID)
	if err != nil {
		return nil, err
	}

	forwardedChannelID, err := fields.ParseID(m.ForwardChannelID)
	if err != nil {
		return nil, err
	}

	var content channel.MessageContent
	if len(m.Content) > 0 {
		if err := json.Unmarshal(m.Content, &content); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message content: %w", err)
		}
	}

	var sysMeta fields.JSON
	if len(m.Metadata) > 0 {
		var parseErr error
		sysMeta, parseErr = fields.ParseJSON("system_metadata", m.Metadata)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse system metadata: %w", parseErr)
		}
	}

	msgType, err := channel.ParseMessageType(m.Type)
	if err != nil {
		return nil, err
	}

	return channel.ReconstituteMessage(
		id,
		channelID,
		authorID,
		msgType,
		content,
		sysMeta,
		replyToMessageID,
		forwardedMessageID,
		forwardedChannelID,
		fields.NewTimestamp(m.PinnedAt),
		fields.NewTimestamp(m.CreatedAt),
		fields.NewTimestamp(m.UpdatedAt),
		fields.NewTimestamp(m.EditedAt),
	), nil
}

func parseMessage(m *channel.Message) (Message, error) {
	contentBytes, err := json.Marshal(m.Content())
	if err != nil {
		return Message{}, fmt.Errorf("failed to marshal message content: %w", err)
	}

	return Message{
		ID:               m.ID().UUID(),
		ChannelID:        m.ChannelID().UUID(),
		AuthorID:         m.AuthorID().UUID(),
		Type:             m.Type().Int(),
		Content:          contentBytes,
		Metadata:         m.Metadata().Bytes(),
		ReplyToMessageID: m.ReplyToMessageID().UUID(),
		ForwardMessageID: m.ForwardMessageID().UUID(),
		ForwardChannelID: m.ForwardChannelID().UUID(),
		PinnedAt:         m.PinnedAt().Time(),
		CreatedAt:        m.CreatedAt().Time(),
		UpdatedAt:        m.UpdatedAt().Time(),
		EditedAt:         m.EditedAt().Time(),
	}, nil
}

type Session struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	RefreshTokenHash []byte     `json:"refresh_token_hash"`
	ClientIP         string     `json:"client_ip"`
	UserAgent        string     `json:"user_agent"`
	OS               string     `json:"os"`
	Client           string     `json:"client"`
	ExpiresAt        time.Time  `json:"expires_at"`
	LastSeenAt       time.Time  `json:"last_seen_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (s Session) ToDomain() (*session.Session, error) {
	id, err := fields.ParseRequiredID("id", s.ID)
	if err != nil {
		return nil, err
	}

	userID, err := fields.ParseRequiredID("user_id", s.UserID)
	if err != nil {
		return nil, err
	}

	refreshTokenHash, err := fields.ParseTokenHash("refresh_token_hash", s.RefreshTokenHash)
	if err != nil {
		return nil, err
	}

	clientIP, err := fields.ParseIP("client_ip", s.ClientIP)
	if err != nil {
		return nil, err
	}

	userAgent, err := fields.ParseUserAgent("user_agent", s.UserAgent)
	if err != nil {
		return nil, err
	}

	osField, err := fields.ParseOS("os", s.OS)
	if err != nil {
		return nil, err
	}

	clientField, err := fields.ParseClient("client", s.Client)
	if err != nil {
		return nil, err
	}

	return session.Reconstitute(
		id,
		userID,
		refreshTokenHash,
		clientIP,
		userAgent,
		osField,
		clientField,
		fields.NewTimestamp(s.ExpiresAt),
		fields.NewTimestamp(s.LastSeenAt),
		fields.NewTimestamp(ptr.From(s.RevokedAt)),
		fields.NewTimestamp(s.CreatedAt),
		fields.NewTimestamp(s.UpdatedAt),
	), nil
}

func parseSession(sess *session.Session) Session {
	if sess == nil {
		return Session{}
	}

	var revokedAt *time.Time
	if r := sess.RevokedAt(); r.IsValid() {
		t := r.Time()
		revokedAt = &t
	}

	return Session{
		ID:               sess.ID().UUID(),
		UserID:           sess.UserID().UUID(),
		RefreshTokenHash: sess.RefreshTokenHash().Bytes.Bytes(),
		ClientIP:         sess.ClientIP().String(),
		UserAgent:        sess.UserAgent().String(),
		OS:               sess.OS().String(),
		Client:           sess.Client().String(),
		ExpiresAt:        sess.ExpiresAt().Time(),
		LastSeenAt:       sess.LastSeenAt().Time(),
		RevokedAt:        revokedAt,
		CreatedAt:        sess.CreatedAt().Time(),
		UpdatedAt:        sess.UpdatedAt().Time(),
	}
}

func marshalSession(sess *session.Session) ([]byte, error) {
	if sess == nil {
		return nil, nil
	}

	dto := parseSession(sess)
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
	return dto.ToDomain()
}

type User struct {
	ID                     uuid.UUID `json:"id"`
	Email                  string    `json:"email"`
	Username               string    `json:"username"`
	DisplayName            string    `json:"display_name"`
	Phone                  string    `json:"phone"`
	Bio                    string    `json:"bio"`
	AvatarURL              string    `json:"avatar_url"`
	BannerColor            string    `json:"banner_color"`
	PreferredPresence      int       `json:"preferred_presence"`
	PreferredPresenceUntil time.Time `json:"preferred_presence_until"`
	VerifiedAt             time.Time `json:"verified_at"`
	DisabledAt             time.Time `json:"disabled_at"`
	DeleteScheduledAt      time.Time `json:"delete_scheduled_at"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

func (u User) ToDomain() (*user.User, error) {
	id, err := fields.ParseRequiredID("id", u.ID)
	if err != nil {
		return nil, err
	}
	email, err := user.ParseEmail("email", u.Email)
	if err != nil {
		return nil, err
	}
	username, err := user.ParseUsername("username", u.Username)
	if err != nil {
		return nil, err
	}
	displayName, err := user.ParseDisplayName("display_name", u.DisplayName)
	if err != nil {
		return nil, err
	}
	phone, err := user.ParsePhone("phone", u.Phone)
	if err != nil {
		return nil, err
	}
	bio, err := user.ParseBio("bio", u.Bio)
	if err != nil {
		return nil, err
	}
	avatarURL, err := fields.ParseURL("avatar_url", u.AvatarURL)
	if err != nil {
		return nil, err
	}
	bannerColor, err := fields.ParseHexColor("banner_color", u.BannerColor)
	if err != nil {
		return nil, err
	}
	prefPresence, err := user.ParsePreferredPresenceFromInt("preferred_presence", u.PreferredPresence)
	if err != nil {
		return nil, err
	}

	return user.Reconstitute(
		id,
		email,
		username,
		user.PasswordHash{},
		phone,
		displayName,
		bio,
		avatarURL,
		bannerColor,
		prefPresence,
		fields.NewTimestamp(u.PreferredPresenceUntil),
		fields.NewTimestamp(u.VerifiedAt),
		fields.NewTimestamp(u.DisabledAt),
		fields.NewTimestamp(u.DeleteScheduledAt),
		fields.NewTimestamp(u.CreatedAt),
		fields.NewTimestamp(u.UpdatedAt),
	), nil
}

func parseUser(u *user.User) User {
	if u == nil {
		return User{}
	}

	return User{
		ID:                     u.ID().UUID(),
		Email:                  u.Email().String(),
		Username:               u.Username().String(),
		DisplayName:            u.DisplayName().String(),
		Phone:                  u.Phone().String(),
		Bio:                    u.Bio().String(),
		AvatarURL:              u.AvatarURL().String(),
		BannerColor:            u.BannerColor().String(),
		PreferredPresence:      u.PreferredPresence().Presence().Int(),
		PreferredPresenceUntil: u.PreferredPresenceUntil().Time(),
		VerifiedAt:             u.VerifiedAt().Time(),
		DisabledAt:             u.DisabledAt().Time(),
		DeleteScheduledAt:      u.DeleteScheduledAt().Time(),
		CreatedAt:              u.CreatedAt().Time(),
		UpdatedAt:              u.UpdatedAt().Time(),
	}
}

func marshalUser(usr *user.User) ([]byte, error) {
	if usr == nil {
		return nil, nil
	}

	dto := parseUser(usr)
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
	return dto.ToDomain()
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
	return dto.ToDomain()
}

func unmarshalMessage(data []byte) (*channel.Message, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var dto Message
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}
	return dto.ToDomain()
}

func parsePresence(val string) presence.Presence {
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return presence.NewOffline()
	}

	p, err := presence.Parse(parsed)
	if err != nil {
		return presence.NewOffline()
	}

	return p
}
