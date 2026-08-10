package fields

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"

	"bonfire-api/internal/sanitize"

	"github.com/google/uuid"
)

// ============================================================================
// HexColor
// ============================================================================

var (
	ErrHexColorInvalid = errors.New("must be a valid hex code (e.g., #FF5733)")
	rgxHexColor        = regexp.MustCompile(`(?i)^#[0-9a-f]{6}$`)
)

type HexColor struct {
	value string
}

func NewHexColor(raw string) (HexColor, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return HexColor{}, nil
	}
	if !rgxHexColor.MatchString(s) {
		return HexColor{}, ErrHexColorInvalid
	}
	return HexColor{value: strings.ToUpper(s)}, nil
}

func (hc HexColor) String() string {
	return hc.value
}

func (hc HexColor) StringPtr() *string {
	if hc.value == "" {
		return nil
	}
	return &hc.value
}

func (hc HexColor) IsValid() bool {
	return hc.value != ""
}

func (hc HexColor) Equals(other HexColor) bool {
	return hc.value == other.value
}

func (hc *HexColor) UnmarshalText(text []byte) (err error) {
	*hc, err = UnmarshalText(text, NewHexColor)
	return err
}

// ============================================================================
// ID
// ============================================================================

var (
	ErrIDNil     = errors.New("id cannot be nil or zero-value")
	ErrIDInvalid = errors.New("invalid uuid format")
)

type ID uuid.UUID

func NewID(raw uuid.UUID) (ID, error) {
	if raw == uuid.Nil {
		return ID{}, ErrIDNil
	}
	return ID(raw), nil
}

func ParseID(raw string) (ID, error) {
	parsed, err := uuid.Parse(sanitize.Text(raw))
	if err != nil {
		return ID{}, ErrIDInvalid
	}
	return NewID(parsed)
}

func (id ID) UUID() uuid.UUID {
	return uuid.UUID(id)
}

func (id ID) String() string {
	return uuid.UUID(id).String()
}

func (id ID) StringPtr() *string {
	if !id.IsValid() {
		return nil
	}
	s := uuid.UUID(id).String()
	return &s
}

func (id ID) IsValid() bool {
	return uuid.UUID(id) != uuid.Nil
}

func (id ID) Equals(other ID) bool {
	return id == other
}

func (id *ID) UnmarshalText(text []byte) (err error) {
	*id, err = UnmarshalText(text, ParseID)
	return err
}

// ============================================================================
// Timestamp
// ============================================================================

var ErrTimestampInvalid = errors.New("timestamp must be a valid RFC 3339 date-time format")

type Timestamp struct {
	value time.Time
}

func NewTimestamp(raw string) (Timestamp, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return Timestamp{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return Timestamp{}, ErrTimestampInvalid
	}
	return NewTimestampFromTime(parsed), nil
}

func NewTimestampFromTime(t time.Time) Timestamp {
	if t.IsZero() {
		return Timestamp{}
	}
	return Timestamp{value: t.UTC()}
}

func NewTimestampFromUnix(sec int64) Timestamp {
	if sec <= 0 {
		return Timestamp{}
	}
	return NewTimestampFromTime(time.Unix(sec, 0))
}

func (t Timestamp) Time() time.Time {
	return t.value
}

func (t Timestamp) TimePtr() *time.Time {
	if !t.IsValid() {
		return nil
	}
	utc := t.value
	return &utc
}

func (t Timestamp) String() string {
	if !t.IsValid() {
		return ""
	}
	return t.value.Format(time.RFC3339)
}

func (t Timestamp) StringPtr() *string {
	if !t.IsValid() {
		return nil
	}
	s := t.String()
	return &s
}

func (t Timestamp) Unix() int64 {
	if !t.IsValid() {
		return 0
	}
	return t.value.Unix()
}

func (t Timestamp) UnixPtr() *int64 {
	if !t.IsValid() {
		return nil
	}
	v := t.Unix()
	return &v
}

func (t Timestamp) IsValid() bool {
	return !t.value.IsZero()
}

func (t Timestamp) Equals(other Timestamp) bool {
	return t.value.Equal(other.value)
}

func (t Timestamp) HasPassed(now time.Time) bool {
	if !t.IsValid() || now.IsZero() {
		return false
	}
	return now.After(t.value)
}

func (t *Timestamp) UnmarshalText(text []byte) (err error) {
	*t, err = UnmarshalText(text, NewTimestamp)
	return err
}

// ============================================================================
// URL
// ============================================================================

var (
	ErrURLInvalid = errors.New("url must be a valid HTTP or HTTPS address")
	ErrURLTooLong = errors.New("url cannot exceed 2048 characters")
)

type URL struct {
	value string
}

func NewURL(raw string) (URL, error) {
	s := sanitize.URL(raw)
	if s == "" {
		return URL{}, nil
	}
	if len(s) > 2048 {
		return URL{}, ErrURLTooLong
	}
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return URL{}, ErrURLInvalid
	}
	parsed, err := url.ParseRequestURI(s)
	if err != nil || parsed.Host == "" {
		return URL{}, ErrURLInvalid
	}
	return URL{value: parsed.String()}, nil
}

func (u URL) String() string {
	return u.value
}

func (u URL) StringPtr() *string {
	if u.value == "" {
		return nil
	}
	return &u.value
}

func (u URL) IsValid() bool {
	return u.value != ""
}

func (u URL) Equals(other URL) bool {
	return u.value == other.value
}

func (u *URL) UnmarshalText(text []byte) (err error) {
	*u, err = UnmarshalText(text, NewURL)
	return err
}

// ============================================================================
// VerificationCode
// ============================================================================

var ErrInvalidVerificationCode = errors.New("verification code must be 6 alphanumeric characters")

var rgxVerificationCode = regexp.MustCompile(`^[2-9A-HJ-NP-Z]{6}$`)

type VerificationCode struct {
	value string
}

func NewVerificationCode(raw string) (VerificationCode, error) {
	s := strings.ToUpper(sanitize.Text(raw))
	if !rgxVerificationCode.MatchString(s) {
		return VerificationCode{}, ErrInvalidVerificationCode
	}
	return VerificationCode{value: s}, nil
}

func (c VerificationCode) String() string {
	return c.value
}

func (c VerificationCode) Equals(other VerificationCode) bool {
	return c.value == other.value
}

// ============================================================================
// Helpers
// ============================================================================

// Go Struct $\rightarrow$ JSON/Bytes = Marshaling (packing up from Go)

// MarshalText is a generic helper to eliminate boilerplate across MarshalText implementations.
func MarshalText[T any](val T, getter func(T) string) ([]byte, error) {
	return []byte(getter(val)), nil
}

// JSON/Bytes -> Go Struct = Unmarshaling (unpacking into Go)

// UnmarshalText is a generic helper to eliminate boilerplate across UnmarshalText implementations
func UnmarshalText[T any](text []byte, parser func(string) (T, error)) (T, error) {
	if len(text) == 0 {
		var zero T
		return zero, nil
	}
	return parser(string(text))
}
