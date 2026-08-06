package user

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"bonfire-api/internal/presence"
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

// HexColor is a color formatted as (?i)^#[0-9a-f]{6}$
type HexColor struct {
	value *string
}

func NewHexColor(raw *string) (HexColor, error) {
	if raw == nil {
		return HexColor{value: nil}, nil
	}

	s := sanitize.Text(*raw)
	if s == "" {
		return HexColor{value: nil}, nil
	}

	if !rgxHexColor.MatchString(s) {
		return HexColor{}, ErrHexColorInvalid
	}

	upper := strings.ToUpper(s)
	return HexColor{value: &upper}, nil
}

func ParseHexColor(raw string) (HexColor, error) {
	return NewHexColor(&raw)
}

func (bc HexColor) String() string {
	if bc.value == nil {
		return ""
	}
	return *bc.value
}

func (bc HexColor) NilString() *string {
	return bc.value
}

func (bc HexColor) IsValid() bool {
	return bc.value != nil && *bc.value != ""
}

func (bc HexColor) Equals(other HexColor) bool {
	if bc.value == nil && other.value == nil {
		return true
	}
	if bc.value == nil || other.value == nil {
		return false
	}
	return *bc.value == *other.value
}

func (bc *HexColor) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*bc = HexColor{value: nil}
		return nil
	}

	raw := string(text)
	parsed, err := NewHexColor(&raw)
	if err != nil {
		return err
	}

	*bc = parsed
	return nil
}

// ============================================================================
// Bio
// ============================================================================

var (
	ErrBioTooLong = errors.New("bio cannot exceed 190 characters")
)

type Bio struct {
	value *string
}

func NewBio(raw *string) (Bio, error) {
	if raw == nil {
		return Bio{value: nil}, nil
	}

	s := sanitize.Text(*raw)
	if s == "" {
		return Bio{value: nil}, nil
	}

	if len([]rune(s)) > 190 {
		return Bio{}, ErrBioTooLong
	}

	return Bio{value: &s}, nil
}

func ParseBio(raw string) (Bio, error) {
	return NewBio(&raw)
}

func (b Bio) String() string {
	if b.value == nil {
		return ""
	}
	return *b.value
}

func (b Bio) NilString() *string {
	return b.value
}

func (b Bio) IsValid() bool {
	return b.value != nil && *b.value != ""
}

func (b Bio) Equals(other Bio) bool {
	if b.value == nil && other.value == nil {
		return true
	}
	if b.value == nil || other.value == nil {
		return false
	}
	return *b.value == *other.value
}

func (b *Bio) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*b = Bio{value: nil}
		return nil
	}

	raw := string(text)
	parsed, err := NewBio(&raw)
	if err != nil {
		return err
	}

	*b = parsed
	return nil
}

// ============================================================================
// DisplayName
// ============================================================================

var (
	ErrDisplayNameEmpty   = errors.New("display name cannot be empty")
	ErrDisplayNameTooLong = errors.New("display name cannot exceed 32 characters")
)

type DisplayName struct {
	value string
}

func NewDisplayName(raw string) (DisplayName, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return DisplayName{}, ErrDisplayNameEmpty
	}
	if len([]rune(s)) > 32 {
		return DisplayName{}, ErrDisplayNameTooLong
	}

	return DisplayName{value: s}, nil
}

func ParseDisplayName(raw string) (DisplayName, error) {
	return NewDisplayName(raw)
}

func (d DisplayName) String() string {
	return d.value
}

func (d DisplayName) NilString() *string {
	if d.value == "" {
		return nil
	}
	return &d.value
}

func (d DisplayName) IsValid() bool {
	return d.value != ""
}

func (d DisplayName) Equals(other DisplayName) bool {
	return d.value == other.value
}

func (d *DisplayName) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*d = DisplayName{}
		return nil
	}

	parsed, err := NewDisplayName(string(text))
	if err != nil {
		return err
	}

	*d = parsed
	return nil
}

// ============================================================================
// Email
// ============================================================================

var (
	ErrEmailEmpty   = errors.New("email cannot be empty")
	ErrEmailTooLong = errors.New("email cannot exceed 255 characters")
	ErrEmailInvalid = errors.New("must be a valid email address")
)

type Email struct {
	value string
}

func NewEmail(raw string) (Email, error) {
	s := sanitize.Email(raw)
	if s == "" {
		return Email{}, ErrEmailEmpty
	}
	if len(s) > 255 {
		return Email{}, ErrEmailTooLong
	}

	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		return Email{}, ErrEmailInvalid
	}

	domain := s[at+1:]
	dot := strings.IndexByte(domain, '.')
	if dot <= 0 || dot == len(domain)-1 {
		return Email{}, ErrEmailInvalid
	}

	return Email{value: s}, nil
}

func ParseEmail(raw string) (Email, error) {
	return NewEmail(raw)
}

func (e Email) String() string {
	return e.value
}

func (e Email) NilString() *string {
	if e.value == "" {
		return nil
	}
	return &e.value
}

func (e Email) IsValid() bool {
	return e.value != ""
}

func (e Email) Equals(other Email) bool {
	return e.value == other.value
}

func (e *Email) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*e = Email{}
		return nil
	}

	parsed, err := NewEmail(string(text))
	if err != nil {
		return err
	}

	*e = parsed
	return nil
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
	parsed, err := uuid.Parse(strings.TrimSpace(raw))
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

func (id ID) IsValid() bool {
	return uuid.UUID(id) != uuid.Nil
}

func (id ID) Equals(other ID) bool {
	return id == other
}

func (id *ID) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*id = ID{}
		return nil
	}

	parsed, err := ParseID(string(text))
	if err != nil {
		return err
	}

	*id = parsed
	return nil
}

// ============================================================================
// Password
// ============================================================================

var (
	ErrPasswordEmpty    = errors.New("password cannot be empty")
	ErrPasswordTooShort = errors.New("password must be at least 12 characters")
	ErrPasswordTooLong  = errors.New("password cannot exceed 255 characters")
)

type Password struct {
	value string
}

func NewPassword(raw string) (Password, error) {
	if raw == "" {
		return Password{}, ErrPasswordEmpty
	}
	if len(raw) < 12 {
		return Password{}, ErrPasswordTooShort
	}
	if len(raw) > 255 {
		return Password{}, ErrPasswordTooLong
	}

	return Password{value: raw}, nil
}

func ParsePassword(raw string) (Password, error) {
	return NewPassword(raw)
}

func (p Password) String() string {
	return p.value
}

func (p Password) NilString() *string {
	if p.value == "" {
		return nil
	}
	return &p.value
}

func (p Password) IsValid() bool {
	return p.value != ""
}

func (p Password) Equals(other Password) bool {
	return p.value == other.value
}

func (p *Password) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*p = Password{}
		return nil
	}

	parsed, err := NewPassword(string(text))
	if err != nil {
		return err
	}

	*p = parsed
	return nil
}

// ============================================================================
// Phone
// ============================================================================

var (
	ErrPhoneInvalid = errors.New("phone must be in international format (e.g., +1234567890)")
	rgxPhone        = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
)

type Phone struct {
	value *string
}

func NewPhone(raw *string) (Phone, error) {
	if raw == nil {
		return Phone{value: nil}, nil
	}

	s := strings.TrimSpace(*raw)
	if s == "" {
		return Phone{value: nil}, nil
	}

	if !rgxPhone.MatchString(s) {
		return Phone{}, ErrPhoneInvalid
	}

	return Phone{value: &s}, nil
}

func ParsePhone(raw string) (Phone, error) {
	return NewPhone(&raw)
}

func (p Phone) String() string {
	if p.value == nil {
		return ""
	}
	return *p.value
}

func (p Phone) NilString() *string {
	return p.value
}

func (p Phone) IsValid() bool {
	return p.value != nil && *p.value != ""
}

func (p Phone) Equals(other Phone) bool {
	if p.value == nil && other.value == nil {
		return true
	}
	if p.value == nil || other.value == nil {
		return false
	}
	return *p.value == *other.value
}

func (p *Phone) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*p = Phone{value: nil}
		return nil
	}

	raw := string(text)
	parsed, err := NewPhone(&raw)
	if err != nil {
		return err
	}

	*p = parsed
	return nil
}

// ============================================================================
// PreferredPresence
// ============================================================================

var (
	ErrPreferredPresenceInvalid = errors.New("invalid preferred presence state")
)

type PreferredPresence struct {
	value *presence.Presence
}

func NewPreferredPresence(raw *string) (PreferredPresence, error) {
	if raw == nil {
		return PreferredPresence{value: nil}, nil
	}

	s := strings.TrimSpace(*raw)
	if s == "" {
		return PreferredPresence{value: nil}, nil
	}

	p, err := presence.New(s)
	if err != nil {
		return PreferredPresence{}, ErrPreferredPresenceInvalid
	}

	switch p {
	case presence.PresenceIdle, presence.PresenceBusy, presence.PresenceDND:
		return PreferredPresence{value: &p}, nil
	default:
		return PreferredPresence{}, ErrPreferredPresenceInvalid
	}
}

func NewPreferredPresenceFromInt16(v int16) (PreferredPresence, error) {
	if v == 0 {
		return PreferredPresence{value: nil}, nil
	}

	p, err := presence.FromInt16(v)
	if err != nil {
		return PreferredPresence{}, ErrPreferredPresenceInvalid
	}

	switch p {
	case presence.PresenceIdle, presence.PresenceBusy, presence.PresenceDND:
		return PreferredPresence{value: &p}, nil
	default:
		return PreferredPresence{}, ErrPreferredPresenceInvalid
	}
}

func ParsePreferredPresence(raw string) (PreferredPresence, error) {
	return NewPreferredPresence(&raw)
}

func (pp PreferredPresence) String() string {
	if pp.value == nil {
		return ""
	}
	return string(*pp.value)
}

func (p PreferredPresence) Int16() int16 {
	return p.value.Int16()
}

func (pp PreferredPresence) NilPresence() *presence.Presence {
	return pp.value
}

func (pp PreferredPresence) IsValid() bool {
	return pp.value != nil
}

func (pp PreferredPresence) Equals(other PreferredPresence) bool {
	if pp.value == nil && other.value == nil {
		return true
	}
	if pp.value == nil || other.value == nil {
		return false
	}
	return *pp.value == *other.value
}

func (pp *PreferredPresence) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*pp = PreferredPresence{value: nil}
		return nil
	}

	raw := string(text)
	parsed, err := NewPreferredPresence(&raw)
	if err != nil {
		return err
	}

	*pp = parsed
	return nil
}

// ============================================================================
// Timestamp
// ============================================================================

var (
	ErrTimestampInvalid = errors.New("timestamp must be a valid RFC 3339 date-time format")
)

type Timestamp struct {
	value *time.Time
}

func NewTimestamp(raw *string) (Timestamp, error) {
	if raw == nil {
		return Timestamp{value: nil}, nil
	}

	s := strings.TrimSpace(*raw)
	if s == "" {
		return Timestamp{value: nil}, nil
	}

	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return Timestamp{}, ErrTimestampInvalid
	}

	utc := parsed.UTC()
	return Timestamp{value: &utc}, nil
}

func NewTimestampFromTime(t *time.Time) Timestamp {
	if t == nil {
		return Timestamp{value: nil}
	}
	utc := t.UTC()
	return Timestamp{value: &utc}
}

func ParseTimestamp(raw string) (Timestamp, error) {
	return NewTimestamp(&raw)
}

func (t Timestamp) Time() *time.Time {
	return t.value
}

func (t Timestamp) Unix() *int64 {
	if t.value == nil {
		return nil
	}
	unix := t.value.Unix()
	return &unix
}

func (t Timestamp) String() string {
	if t.value == nil {
		return ""
	}
	return t.value.Format(time.RFC3339)
}

func (t Timestamp) NilString() *string {
	if t.value == nil {
		return nil
	}
	s := t.value.Format(time.RFC3339)
	return &s
}

func (t Timestamp) IsValid() bool {
	return t.value != nil
}

func (t Timestamp) Equals(other Timestamp) bool {
	if t.value == nil && other.value == nil {
		return true
	}
	if t.value == nil || other.value == nil {
		return false
	}
	return t.value.Equal(*other.value)
}

func (t *Timestamp) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*t = Timestamp{value: nil}
		return nil
	}

	raw := string(text)
	parsed, err := NewTimestamp(&raw)
	if err != nil {
		return err
	}

	*t = parsed
	return nil
}

// ============================================================================
// URL
// ============================================================================

var (
	ErrURLInvalid = errors.New("url must be a valid HTTP or HTTPS address")
	ErrURLTooLong = errors.New("url cannot exceed 2048 characters")
)

type URL struct {
	value *string
}

func NewURL(raw *string) (URL, error) {
	if raw == nil {
		return URL{value: nil}, nil
	}

	s := strings.TrimSpace(*raw)
	if s == "" {
		return URL{value: nil}, nil
	}

	if len(s) > 2048 {
		return URL{}, ErrURLTooLong
	}

	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return URL{}, ErrURLInvalid
	}

	return URL{value: &s}, nil
}

func ParseURL(raw string) (URL, error) {
	return NewURL(&raw)
}

func (u URL) String() string {
	if u.value == nil {
		return ""
	}
	return *u.value
}

func (u URL) NilString() *string {
	return u.value
}

func (u URL) IsValid() bool {
	return u.value != nil && *u.value != ""
}

func (u URL) Equals(other URL) bool {
	if u.value == nil && other.value == nil {
		return true
	}
	if u.value == nil || other.value == nil {
		return false
	}
	return *u.value == *other.value
}

func (u *URL) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*u = URL{value: nil}
		return nil
	}

	raw := string(text)
	parsed, err := NewURL(&raw)
	if err != nil {
		return err
	}

	*u = parsed
	return nil
}

// ============================================================================
// Username
// ============================================================================

var (
	ErrUsernameEmpty      = errors.New("username cannot be empty")
	ErrUsernameTooShort   = errors.New("username must be at least 3 characters")
	ErrUsernameTooLong    = errors.New("username cannot exceed 32 characters")
	ErrUsernameInvalidFmt = errors.New("username must start and end with a letter/number and use only letters, numbers, and non-consecutive dots or underscores")
	ErrUsernameReserved   = errors.New("this username is reserved and cannot be used")

	rgxUsername = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9_.]?[a-zA-Z0-9])+$`)

	reservedUsernames = map[string]bool{
		"admin":     true,
		"root":      true,
		"support":   true,
		"system":    true,
		"moderator": true,
		"bonfire":   true,
	}
)

type Username struct {
	value string
}

func NewUsername(raw string) (Username, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return Username{}, ErrUsernameEmpty
	}
	if len(s) < 3 {
		return Username{}, ErrUsernameTooShort
	}
	if len(s) > 32 {
		return Username{}, ErrUsernameTooLong
	}
	if !rgxUsername.MatchString(s) {
		return Username{}, ErrUsernameInvalidFmt
	}
	if reservedUsernames[strings.ToLower(s)] {
		return Username{}, ErrUsernameReserved
	}

	return Username{value: s}, nil
}

func ParseUsername(raw string) (Username, error) {
	return NewUsername(raw)
}

func (u Username) String() string {
	return u.value
}

func (u Username) NilString() *string {
	if u.value == "" {
		return nil
	}
	return &u.value
}

func (u Username) IsValid() bool {
	return u.value != ""
}

func (u Username) Equals(other Username) bool {
	return u.value == other.value
}

func (u *Username) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*u = Username{}
		return nil
	}

	parsed, err := NewUsername(string(text))
	if err != nil {
		return err
	}

	*u = parsed
	return nil
}
