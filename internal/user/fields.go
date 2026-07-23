package user

import (
	"errors"
	"regexp"
	"strings"

	"bonfire-api/internal/presence"
	"bonfire-api/internal/sanitize"
)

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
	if at <= 0 || at == len(s)-1 || strings.IndexByte(s[at+1:], '.') <= 0 {
		return Email{}, ErrEmailInvalid
	}

	return Email{value: s}, nil
}

func (e Email) String() string {
	return e.value
}

func (e Email) IsValid() bool {
	return e.value != ""
}

func (e *Email) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return nil
	}
	parsed, err := NewEmail(string(text))
	if err != nil {
		return err
	}
	*e = parsed
	return nil
}

var (
	ErrUsernameEmpty      = errors.New("username cannot be empty")
	ErrUsernameTooShort   = errors.New("username must be at least 3 characters")
	ErrUsernameTooLong    = errors.New("username cannot exceed 32 characters")
	ErrUsernameInvalidFmt = errors.New("username must start and end with a letter/number and use only letters, numbers, and non-consecutive dots or underscores")

	rgxUsername = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9_.]?[a-zA-Z0-9])+$`)
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

	return Username{value: s}, nil
}

func (u Username) String() string {
	return u.value
}

func (u Username) Equals(other Username) bool {
	return u.value == other.value
}

func (u Username) IsValid() bool {
	return u.value != ""
}

func (u *Username) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return nil
	}
	parsed, err := NewUsername(string(text))
	if err != nil {
		return err
	}
	*u = parsed
	return nil
}

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

func (p *Password) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return nil
	}
	parsed, err := NewPassword(string(text))
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}

func NewPreferredPresence(raw string) (*presence.Presence, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, nil
	}

	p, err := presence.New(s)
	if err != nil {
		return nil, presence.ErrInvalidPresence
	}

	switch p {
	case presence.PresenceIdle, presence.PresenceBusy, presence.PresenceDND:
		return &p, nil
	default:
		return nil, presence.ErrInvalidPresence
	}
}

var (
	ErrProfileDisplayNameEmpty    = errors.New("display name cannot be empty")
	ErrProfileDisplayNameTooShort = errors.New("display name must be at least 3 characters")
	ErrProfileDisplayNameTooLong  = errors.New("display name cannot exceed 32 characters")
)

type ProfileDisplayName struct {
	value string
}

func NewProfileDisplayName(raw string) (ProfileDisplayName, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return ProfileDisplayName{}, ErrProfileDisplayNameEmpty
	}
	if len(s) < 3 {
		return ProfileDisplayName{}, ErrProfileDisplayNameTooShort
	}
	if len(s) > 32 {
		return ProfileDisplayName{}, ErrProfileDisplayNameTooLong
	}

	return ProfileDisplayName{value: s}, nil
}

func (d ProfileDisplayName) String() string {
	return d.value
}

func (d ProfileDisplayName) IsValid() bool {
	return d.value != ""
}

func (d *ProfileDisplayName) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return nil
	}
	parsed, err := NewProfileDisplayName(string(text))
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}
