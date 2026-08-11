package user

import (
	"regexp"
	"strings"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/sanitize"
)

// ============================================================================
// Bio
// ============================================================================

const MaxBioLength = 190

type Bio struct {
	fields.Text
}

func ParseBio(field, raw string) (Bio, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return Bio{}, nil
	}

	if err := fields.Validate(field, s, fields.ValidateCfg{
		MaxLen:    MaxBioLength,
		IsRuneLen: true,
	}); err != nil {
		return Bio{}, err
	}

	return Bio{Text: fields.NewText(s)}, nil
}

func (b Bio) Equals(other Bio) bool {
	return b.Text.Equals(other.Text)
}

func (b *Bio) UnmarshalText(text []byte) error {
	var err error
	*b, err = fields.UnmarshalText(text, "bio", ParseBio)
	return err
}

// ============================================================================
// DisplayName
// ============================================================================

const MaxDisplayNameLength = 32

type DisplayName struct {
	fields.Text
}

func ParseDisplayName(field, raw string) (DisplayName, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return DisplayName{}, fields.ErrRequired(field)
	}

	if err := fields.Validate(field, s, fields.ValidateCfg{
		MaxLen:    MaxDisplayNameLength,
		IsRuneLen: true,
	}); err != nil {
		return DisplayName{}, err
	}

	return DisplayName{Text: fields.NewText(s)}, nil
}

func (d DisplayName) Equals(other DisplayName) bool {
	return d.Text.Equals(other.Text)
}

func (d *DisplayName) UnmarshalText(text []byte) error {
	var err error
	*d, err = fields.UnmarshalText(text, "display_name", ParseDisplayName)
	return err
}

// ============================================================================
// Email
// ============================================================================

const MaxEmailLength = 255

var rgxEmail = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

type Email struct {
	fields.Text
}

func ParseEmail(field, raw string) (Email, error) {
	s := sanitize.Email(raw)
	if s == "" {
		return Email{}, fields.ErrRequired(field)
	}

	if err := fields.Validate(field, s, fields.ValidateCfg{
		MaxLen: MaxEmailLength,
		Regex:  rgxEmail,
	}); err != nil {
		return Email{}, err
	}

	return Email{Text: fields.NewText(s)}, nil
}

func (e Email) Equals(other Email) bool {
	return e.Text.Equals(other.Text)
}

func (e *Email) UnmarshalText(text []byte) error {
	var err error
	*e, err = fields.UnmarshalText(text, "email", ParseEmail)
	return err
}

// ============================================================================
// Password
// ============================================================================

const (
	MinPasswordLength = 12
	MaxPasswordLength = 255
)

type Password struct {
	fields.Text
}

func ParsePassword(field, raw string) (Password, error) {
	if raw == "" {
		return Password{}, fields.ErrRequired(field)
	}

	if err := fields.Validate(field, raw, fields.ValidateCfg{
		MinLen: MinPasswordLength,
		MaxLen: MaxPasswordLength,
	}); err != nil {
		return Password{}, err
	}

	return Password{Text: fields.NewText(raw)}, nil
}

func (p Password) Equals(other Password) bool {
	return p.Text.Equals(other.Text)
}

func (p *Password) UnmarshalText(text []byte) error {
	var err error
	*p, err = fields.UnmarshalText(text, "password", ParsePassword)
	return err
}

// ============================================================================
// PasswordHash
// ============================================================================

const (
	MinPasswordHashLength = 50
	MaxPasswordHashLength = 255
)

type PasswordHash struct {
	fields.Text
}

func ParsePasswordHash(field, raw string) (PasswordHash, error) {
	if raw == "" {
		return PasswordHash{}, fields.ErrRequired(field)
	}

	if err := fields.Validate(field, raw, fields.ValidateCfg{
		MinLen: MinPasswordHashLength,
		MaxLen: MaxPasswordHashLength,
	}); err != nil {
		return PasswordHash{}, err
	}

	return PasswordHash{Text: fields.NewText(raw)}, nil
}

func (p PasswordHash) Equals(other PasswordHash) bool {
	return p.Text.Equals(other.Text)
}

func (p *PasswordHash) UnmarshalText(text []byte) error {
	var err error
	*p, err = fields.UnmarshalText(text, "password_hash", ParsePasswordHash)
	return err
}

// ============================================================================
// Phone
// ============================================================================

var rgxPhone = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

type Phone struct {
	fields.Text
}

func ParsePhone(field, raw string) (Phone, error) {
	s := sanitize.Phone(raw)
	if s == "" {
		return Phone{}, nil
	}

	if err := fields.Validate(field, s, fields.ValidateCfg{
		Regex: rgxPhone,
	}); err != nil {
		return Phone{}, fields.ErrInvalidFormat(field, "Phone must be in international E.164 format (e.g., +1234567890)")
	}

	return Phone{Text: fields.NewText(s)}, nil
}

func (p Phone) Equals(other Phone) bool {
	return p.Text.Equals(other.Text)
}

func (p *Phone) UnmarshalText(text []byte) error {
	var err error
	*p, err = fields.UnmarshalText(text, "phone", ParsePhone)
	return err
}

// ============================================================================
// PreferredPresence
// ============================================================================

func ErrPreferredPresenceInvalid(field string) *errs.Error {
	return errs.InvalidArgument("Invalid preferred presence status.").
		Reason("PREFERRED_PRESENCE_INVALID").
		FieldViolation(field, "Must be one of: idle, busy, dnd.", "INVALID_ENUM_VALUE")
}

type PreferredPresence struct {
	value presence.Presence
}

func ParsePreferredPresence(field, raw string) (PreferredPresence, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return PreferredPresence{}, nil
	}

	p, err := presence.ParseBytes([]byte(s))
	if err != nil {
		return PreferredPresence{}, ErrPreferredPresenceInvalid(field)
	}

	return ParsePreferredPresenceFromPresence(field, p)
}

func ParsePreferredPresenceFromInt16(field string, v int16) (PreferredPresence, error) {
	if v == 0 {
		return PreferredPresence{}, nil
	}

	p, err := presence.FromInt16(v)
	if err != nil {
		return PreferredPresence{}, ErrPreferredPresenceInvalid(field)
	}

	return ParsePreferredPresenceFromPresence(field, p)
}

func ParsePreferredPresenceFromPresence(field string, p presence.Presence) (PreferredPresence, error) {
	switch p {
	case presence.PresenceIdle, presence.PresenceBusy, presence.PresenceDND:
		return PreferredPresence{value: p}, nil
	}
	return PreferredPresence{}, ErrPreferredPresenceInvalid(field)
}

func (pp PreferredPresence) Presence() presence.Presence { return pp.value }

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
	str := pp.String()
	return &str
}

func (pp PreferredPresence) Int16() int16 { return pp.value.Int16() }

func (pp PreferredPresence) Int16Ptr() *int16 {
	if !pp.IsSet() {
		return nil
	}
	return pp.value.Int16Ptr()
}

func (pp PreferredPresence) IsZero() bool { return pp.value == presence.PresenceUnknown }

func (pp PreferredPresence) IsValid() bool { return pp.IsZero() || pp.IsSet() }

func (pp PreferredPresence) IsSet() bool {
	switch pp.value {
	case presence.PresenceIdle, presence.PresenceBusy, presence.PresenceDND:
		return true
	default:
		return false
	}
}

func (pp PreferredPresence) Equals(other PreferredPresence) bool {
	return pp.value == other.value
}

func (pp PreferredPresence) MarshalText() ([]byte, error) {
	if !pp.IsSet() {
		return nil, nil
	}
	return []byte(pp.String()), nil
}

func (pp *PreferredPresence) UnmarshalText(text []byte) error {
	var err error
	*pp, err = fields.UnmarshalText(text, "preferred_presence", ParsePreferredPresence)
	return err
}

// ============================================================================
// Username
// ============================================================================

const (
	MinUsernameLength = 3
	MaxUsernameLength = 32
)

var rgxUsername = regexp.MustCompile(`^[a-zA-Z0-9]+(?:[._][a-zA-Z0-9]+)*$`)

type Username struct {
	fields.Text
}

func ParseUsername(field, raw string) (Username, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return Username{}, fields.ErrRequired(field)
	}

	if err := fields.Validate(field, s, fields.ValidateCfg{
		MinLen:    MinUsernameLength,
		MaxLen:    MaxUsernameLength,
		Regex:     rgxUsername,
		IsRuneLen: true,
	}); err != nil {
		return Username{}, err
	}

	switch strings.ToLower(s) {
	case "admin", "root", "support", "system", "moderator", "bonfire":
		return Username{}, errs.InvalidArgument("Invalid username.").
			Reason("USERNAME_RESERVED").
			FieldViolation(field, "This username is reserved and cannot be used", "RESERVED_VALUE")
	}

	return Username{Text: fields.NewText(s)}, nil
}

func (u Username) Equals(other Username) bool {
	return strings.EqualFold(u.String(), other.String())
}

func (u *Username) UnmarshalText(text []byte) error {
	var err error
	*u, err = fields.UnmarshalText(text, "username", ParseUsername)
	return err
}
