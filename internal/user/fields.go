package user

import (
	"errors"
	"regexp"
	"strings"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/sanitize"
)

// ============================================================================
// Bio
// ============================================================================

var ErrBioTooLong = errors.New("bio cannot exceed 190 characters")

type Bio struct {
	value string
}

func NewBio(raw string) (Bio, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return Bio{}, nil
	}
	if len([]rune(s)) > 190 {
		return Bio{}, ErrBioTooLong
	}
	return Bio{value: s}, nil
}

func (b Bio) String() string {
	return b.value
}

func (b Bio) StringPtr() *string {
	if b.value == "" {
		return nil
	}
	return &b.value
}

func (b Bio) IsValid() bool {
	return b.value != ""
}

func (b Bio) Equals(other Bio) bool {
	return b.value == other.value
}

func (b *Bio) UnmarshalText(text []byte) (err error) {
	*b, err = fields.UnmarshalText(text, NewBio)
	return err
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

func (d DisplayName) String() string {
	return d.value
}

func (d DisplayName) StringPtr() *string {
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

func (d *DisplayName) UnmarshalText(text []byte) (err error) {
	*d, err = fields.UnmarshalText(text, NewDisplayName)
	return err
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

func (e Email) String() string {
	return e.value
}

func (e Email) StringPtr() *string {
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

func (e *Email) UnmarshalText(text []byte) (err error) {
	*e, err = fields.UnmarshalText(text, NewEmail)
	return err
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

func (p Password) String() string {
	return p.value
}

func (p Password) StringPtr() *string {
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

func (p *Password) UnmarshalText(text []byte) (err error) {
	*p, err = fields.UnmarshalText(text, NewPassword)
	return err
}

// ============================================================================
// PasswordHash
// ============================================================================

var (
	ErrPasswordHashEmpty    = errors.New("password hash cannot be empty")
	ErrPasswordHashTooShort = errors.New("password hash must be at least 50 characters")
	ErrPasswordHashTooLong  = errors.New("password hash cannot exceed 255 characters")
)

type PasswordHash struct {
	value string
}

func NewPasswordHash(raw string) (PasswordHash, error) {
	if raw == "" {
		return PasswordHash{}, ErrPasswordHashEmpty
	}
	if len(raw) < 50 {
		return PasswordHash{}, ErrPasswordHashTooShort
	}
	if len(raw) > 255 {
		return PasswordHash{}, ErrPasswordHashTooLong
	}
	return PasswordHash{value: raw}, nil
}

func (p PasswordHash) String() string {
	return p.value
}

func (p PasswordHash) StringPtr() *string {
	if p.value == "" {
		return nil
	}
	return &p.value
}

func (p PasswordHash) IsValid() bool {
	return p.value != ""
}

func (p PasswordHash) Equals(other PasswordHash) bool {
	return p.value == other.value
}

func (p *PasswordHash) UnmarshalText(text []byte) (err error) {
	*p, err = fields.UnmarshalText(text, NewPasswordHash)
	return err
}

// ============================================================================
// Phone
// ============================================================================

var (
	ErrPhoneInvalid = errors.New("phone must be in international format (e.g., +1234567890)")
	rgxPhone        = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
)

type Phone struct {
	value string
}

func NewPhone(raw string) (Phone, error) {
	s := sanitize.Phone(raw)
	if s == "" {
		return Phone{}, nil
	}
	if !rgxPhone.MatchString(s) {
		return Phone{}, ErrPhoneInvalid
	}
	return Phone{value: s}, nil
}

func (p Phone) String() string {
	return p.value
}

func (p Phone) StringPtr() *string {
	if p.value == "" {
		return nil
	}
	return &p.value
}

func (p Phone) IsValid() bool {
	return p.value != ""
}

func (p Phone) Equals(other Phone) bool {
	return p.value == other.value
}

func (p *Phone) UnmarshalText(text []byte) (err error) {
	*p, err = fields.UnmarshalText(text, NewPhone)
	return err
}

// ============================================================================
// PreferredPresence
// ============================================================================

var ErrPreferredPresenceInvalid = errors.New("invalid preferred presence state")

type PreferredPresence struct {
	value presence.Presence
}

func NewPreferredPresence(raw string) (PreferredPresence, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return PreferredPresence{}, nil
	}

	p, err := presence.New(s)
	if err != nil {
		return PreferredPresence{}, ErrPreferredPresenceInvalid
	}

	return PreferredPresenceFromPresence(p)
}

func PreferredPresenceFromInt16(v int16) (PreferredPresence, error) {
	if v == 0 {
		return PreferredPresence{}, nil
	}
	p, err := presence.FromInt16(v)
	if err != nil {
		return PreferredPresence{}, ErrPreferredPresenceInvalid
	}
	return PreferredPresenceFromPresence(p)
}

func PreferredPresenceFromPresence(p presence.Presence) (PreferredPresence, error) {
	if p == presence.PresenceUnknown {
		return PreferredPresence{}, nil
	}
	switch p {
	case presence.PresenceIdle, presence.PresenceBusy, presence.PresenceDND:
		return PreferredPresence{value: p}, nil
	default:
		return PreferredPresence{}, ErrPreferredPresenceInvalid
	}
}

func (pp PreferredPresence) Presence() presence.Presence {
	return pp.value
}

func (pp PreferredPresence) String() string {
	if !pp.IsSet() {
		return ""
	}
	return pp.value.String()
}

func (pp PreferredPresence) StringPtr() *string {
	if !pp.IsSet() {
		return nil
	}
	s := pp.String()
	return &s
}

func (pp PreferredPresence) Int16() int16 {
	return pp.value.Int16()
}

func (pp PreferredPresence) Int16Ptr() *int16 {
	return pp.value.Int16Ptr()
}

func (pp PreferredPresence) IsValid() bool {
	return pp.value == presence.PresenceUnknown || pp.IsSet()
}

func (pp PreferredPresence) IsSet() bool {
	return pp.value == presence.PresenceIdle ||
		pp.value == presence.PresenceBusy ||
		pp.value == presence.PresenceDND
}

func (pp PreferredPresence) Equals(other PreferredPresence) bool {
	return pp.value == other.value
}

func (pp PreferredPresence) MarshalText() ([]byte, error) {
	if !pp.IsSet() {
		return nil, nil
	}
	return pp.value.MarshalText()
}

func (pp *PreferredPresence) UnmarshalText(text []byte) (err error) {
	*pp, err = fields.UnmarshalText(text, NewPreferredPresence)
	return err
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

	rgxUsername = regexp.MustCompile(`^[a-zA-Z0-9]+(?:[._][a-zA-Z0-9]+)*$`)

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
	s := sanitize.Text(raw)
	if s == "" {
		return Username{}, ErrUsernameEmpty
	}

	runeLen := len([]rune(s))
	if runeLen < 3 {
		return Username{}, ErrUsernameTooShort
	}
	if runeLen > 32 {
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

func (u Username) String() string {
	return u.value
}

func (u Username) StringPtr() *string {
	if u.value == "" {
		return nil
	}
	return &u.value
}

func (u Username) IsValid() bool {
	return u.value != ""
}

func (u Username) Equals(other Username) bool {
	return strings.EqualFold(u.value, other.value)
}

func (u *Username) UnmarshalText(text []byte) (err error) {
	*u, err = fields.UnmarshalText(text, NewUsername)
	return err
}
