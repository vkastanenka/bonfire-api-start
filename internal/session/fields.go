package session

import (
	"fmt"
	"net"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/sanitize"
)

// ============================================================================
// RefreshTokenHash
// ============================================================================

const RefreshTokenHashByteLength = 32

type RefreshTokenHash struct {
	fields.Bytes
}

func ParseRefreshTokenHash(field string, raw []byte) (RefreshTokenHash, error) {
	if len(raw) == 0 {
		return RefreshTokenHash{}, fields.ErrRequired(field)
	}

	if len(raw) != RefreshTokenHashByteLength {
		return RefreshTokenHash{}, fields.ErrInvalidFormat(
			field,
			fmt.Sprintf("Refresh token hash must be exactly %d bytes.", RefreshTokenHashByteLength),
		)
	}

	return RefreshTokenHash{Bytes: fields.NewBytes(raw)}, nil
}

func (h RefreshTokenHash) Equals(other RefreshTokenHash) bool {
	return h.Bytes.Equals(other.Bytes)
}

func (h *RefreshTokenHash) UnmarshalText(text []byte) error {
	var err error
	*h, err = fields.UnmarshalTextBytes(text, "refresh_token_hash", ParseRefreshTokenHash)
	return err
}

// ============================================================================
// ClientIP
// ============================================================================

type ClientIP struct {
	fields.Text
	ip net.IP
}

func ParseClientIP(field, raw string) (ClientIP, error) {
	s := sanitize.Text(raw)
	err := fields.Validate(field, s, fields.ValidateCfg{
		Required: true,
	})
	if err != nil {
		return ClientIP{}, err
	}

	parsedIP := net.ParseIP(s)
	if parsedIP == nil {
		return ClientIP{}, fields.ErrInvalidFormat(field, "Must be a valid IPv4 or IPv6 address")
	}

	return ClientIP{
		Text: fields.NewText(parsedIP.String()),
		ip:   parsedIP,
	}, nil
}

func (c ClientIP) IP() net.IP { return c.ip }

func (c ClientIP) Equals(other ClientIP) bool {
	return c.ip.Equal(other.ip)
}

func (c *ClientIP) UnmarshalText(text []byte) error {
	var err error
	*c, err = fields.UnmarshalText(text, "client_ip", ParseClientIP)
	return err
}

// ============================================================================
// UserAgent
// ============================================================================

const (
	MinUserAgentLength = 1
	MaxUserAgentLength = 1000
)

type UserAgent struct {
	fields.Text
}

func ParseUserAgent(field, raw string) (UserAgent, error) {
	s := sanitize.Text(raw)
	if err := fields.Validate(field, s, fields.ValidateCfg{
		MinLen:   MinUserAgentLength,
		MaxLen:   MaxUserAgentLength,
		Required: true,
	}); err != nil {
		return UserAgent{}, err
	}

	return UserAgent{Text: fields.NewText(s)}, nil
}

func (u UserAgent) Equals(other UserAgent) bool {
	return u.Text.Equals(other.Text)
}

func (u *UserAgent) UnmarshalText(text []byte) error {
	var err error
	*u, err = fields.UnmarshalText(text, "user_agent", ParseUserAgent)
	return err
}

// ============================================================================
// OS
// ============================================================================

const (
	MinOSLength = 1
	MaxOSLength = 100
)

type OS struct {
	fields.Text
}

func ParseOS(field, raw string) (OS, error) {
	s := sanitize.Text(raw)
	if err := fields.Validate(field, s, fields.ValidateCfg{
		MinLen:   MinOSLength,
		MaxLen:   MaxOSLength,
		Required: true,
	}); err != nil {
		return OS{}, err
	}

	return OS{Text: fields.NewText(s)}, nil
}

func (o OS) Equals(other OS) bool {
	return o.Text.Equals(other.Text)
}

func (o *OS) UnmarshalText(text []byte) error {
	var err error
	*o, err = fields.UnmarshalText(text, "os", ParseOS)
	return err
}

// ============================================================================
// Client
// ============================================================================

const (
	MinClientLength = 1
	MaxClientLength = 100
)

type Client struct {
	fields.Text
}

func ParseClient(field, raw string) (Client, error) {
	s := sanitize.Text(raw)
	if err := fields.Validate(field, s, fields.ValidateCfg{
		MinLen:   MinClientLength,
		MaxLen:   MaxClientLength,
		Required: true,
	}); err != nil {
		return Client{}, err
	}

	return Client{Text: fields.NewText(s)}, nil
}

func (c Client) Equals(other Client) bool {
	return c.Text.Equals(other.Text)
}

func (c *Client) UnmarshalText(text []byte) error {
	var err error
	*c, err = fields.UnmarshalText(text, "client", ParseClient)
	return err
}

// ============================================================================
// ExpiresAt
// ============================================================================

type ExpiresAt struct {
	fields.Timestamp
}

func ParseExpiresAt(field string, expiresAt time.Time, now time.Time) (ExpiresAt, error) {
	ts := fields.NewTimestampFromTime(expiresAt)
	if !ts.IsValid() {
		return ExpiresAt{}, fields.ErrRequired(field)
	}

	if !ts.Time().After(now) {
		return ExpiresAt{}, fields.ErrInvalidFormat(field, "ExpiresAt timestamp must be in the future")
	}

	return ExpiresAt{Timestamp: ts}, nil
}

func (e ExpiresAt) Equals(other ExpiresAt) bool {
	return e.Timestamp.Equals(other.Timestamp)
}
