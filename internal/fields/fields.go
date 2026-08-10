package fields

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/sanitize"

	"github.com/google/uuid"
)

// ============================================================================
// Text
// ============================================================================

type Text struct {
	value string
}

func NewText(v string) Text           { return Text{value: v} }
func (t Text) String() string         { return t.value }
func (t Text) IsZero() bool           { return t.value == "" }
func (t Text) IsValid() bool          { return !t.IsZero() }
func (t Text) Equals(other Text) bool { return t.value == other.value }

func (t Text) StringPtr() *string {
	if t.IsZero() {
		return nil
	}
	return ptr.To(t.value)
}

func (t Text) MarshalText() ([]byte, error) {
	return []byte(t.value), nil
}

// ============================================================================
// HexColor
// ============================================================================

var rgxHexColor = regexp.MustCompile(`(?i)^#[0-9a-f]{6}$`)

type HexColor struct{ Text }

func NewHexColor(raw string) (HexColor, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return HexColor{}, nil
	}

	if !rgxHexColor.MatchString(s) {
		return HexColor{}, errs.InvalidArgument("Invalid hex color.").
			Reason("HEX_COLOR_INVALID_FORMAT").
			FieldViolation("hex_color", "Must be a valid hex code (e.g., #FF5733)", "INVALID_FORMAT")
	}

	return HexColor{Text: NewText(strings.ToUpper(s))}, nil
}

func (hc HexColor) Equals(other HexColor) bool { return hc.Text.Equals(other.Text) }

func (hc *HexColor) UnmarshalText(text []byte) (err error) {
	*hc, err = UnmarshalText(text, NewHexColor)
	return err
}

// ============================================================================
// ID
// ============================================================================

type ID uuid.UUID

func NewID(raw uuid.UUID) ID {
	if raw == uuid.Nil {
		return ID{}
	}
	return ID(raw)
}

func ParseID(raw string) (ID, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return ID{}, nil
	}

	parsed, err := uuid.Parse(s)
	if err != nil {
		return ID{}, errs.InvalidArgument("Invalid identifier.").
			Reason("ID_INVALID_FORMAT").
			FieldViolation("id", "Must be a valid UUID", "INVALID_FORMAT")
	}

	return ID(parsed), nil
}

func (id ID) UUID() uuid.UUID      { return uuid.UUID(id) }
func (id ID) String() string       { return uuid.UUID(id).String() }
func (id ID) IsValid() bool        { return uuid.UUID(id) != uuid.Nil }
func (id ID) Equals(other ID) bool { return id == other }

func (id ID) StringPtr() *string {
	if !id.IsValid() {
		return nil
	}
	return ptr.To(id.String())
}

func (id *ID) UnmarshalText(text []byte) (err error) {
	*id, err = UnmarshalText(text, ParseID)
	return err
}

// ============================================================================
// Timestamp
// ============================================================================

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
		return Timestamp{}, errs.InvalidArgument("Invalid timestamp.").
			Reason("TIMESTAMP_INVALID_FORMAT").
			FieldViolation("timestamp", "Timestamp must be a valid RFC 3339 date-time format", "INVALID_FORMAT")
	}

	return Timestamp{value: parsed.UTC()}, nil
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

func (t Timestamp) Time() time.Time         { return t.value }
func (t Timestamp) IsValid() bool           { return !t.value.IsZero() }
func (t Timestamp) Equals(o Timestamp) bool { return t.value.Equal(o.value) }

func (t Timestamp) TimePtr() *time.Time {
	if !t.IsValid() {
		return nil
	}
	return ptr.To(t.value)
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
	return ptr.To(t.String())
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
	return ptr.To(t.Unix())
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

const MaxURLLength = 2048

type URL struct{ Text }

func NewURL(raw string) (URL, error) {
	s := sanitize.URL(raw)
	if s == "" {
		return URL{}, nil
	}

	if len(s) > MaxURLLength {
		return URL{}, errs.InvalidArgument("Invalid URL.").
			Reason("URL_TOO_LONG").
			FieldViolation("url", "URL must not exceed 2048 characters", "MAX_LENGTH_EXCEEDED")
	}

	parsed, err := url.ParseRequestURI(s)
	if err != nil || parsed.Host == "" || (!strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://")) {
		return URL{}, errs.InvalidArgument("Invalid URL.").
			Reason("URL_INVALID_FORMAT").
			FieldViolation("url", "URL must be a valid HTTP or HTTPS address", "INVALID_FORMAT")
	}

	return URL{Text: NewText(parsed.String())}, nil
}

func (u URL) Equals(other URL) bool { return u.Text.Equals(other.Text) }

func (u *URL) UnmarshalText(text []byte) (err error) {
	*u, err = UnmarshalText(text, NewURL)
	return err
}

// ============================================================================
// VerificationCode
// ============================================================================

var rgxVerificationCode = regexp.MustCompile(`^[2-9A-HJ-NP-Z]{6}$`)

type VerificationCode struct{ Text }

func NewVerificationCode(raw string) (VerificationCode, error) {
	s := strings.ToUpper(sanitize.Text(raw))
	if s == "" {
		return VerificationCode{}, nil
	}

	if !rgxVerificationCode.MatchString(s) {
		return VerificationCode{}, errs.InvalidArgument("Invalid verification code.").
			Reason("VERIFICATION_CODE_INVALID_FORMAT").
			FieldViolation("verification_code", "Verification code must be 6 alphanumeric characters", "INVALID_FORMAT")
	}

	return VerificationCode{Text: NewText(s)}, nil
}

func (c VerificationCode) Equals(other VerificationCode) bool { return c.Text.Equals(other.Text) }

func (c *VerificationCode) UnmarshalText(text []byte) (err error) {
	*c, err = UnmarshalText(text, NewVerificationCode)
	return err
}

// ============================================================================
// Helpers
// ============================================================================

func MarshalText[T any](val T, getter func(T) string) ([]byte, error) {
	return []byte(getter(val)), nil
}

func UnmarshalText[T any](text []byte, parser func(string) (T, error)) (T, error) {
	if len(text) == 0 {
		var zero T
		return zero, nil
	}
	return parser(string(text))
}
