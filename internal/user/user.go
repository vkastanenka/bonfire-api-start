package user

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrAlreadyVerified = errors.New("user is already verified")
	ErrInvalidPresence = errors.New("invalid presence state")
	ErrInvalidUsername = errors.New("invalid username format")
)

// User represents the User aggregate root.
type User struct {
	id           uuid.UUID
	email        string
	username     string
	passwordHash string
	presence     Presence
	verifiedAt   *time.Time
	createdAt    time.Time
	updatedAt    time.Time
}

// New creates a fresh User aggregate, enforcing initialization rules.
func New(id uuid.UUID, email, username, passwordHash string, now time.Time) (*User, error) {
	if !IsValidUsername(username) {
		return nil, ErrInvalidUsername
	}

	return &User{
		id:           id,
		email:        email,
		username:     username,
		passwordHash: passwordHash,
		presence:     PresenceOffline,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

// Reconstitute builds a User from storage bypasses creation domain logic.
// This is used by your repository infrastructure layer.
func Reconstitute(
	id uuid.UUID,
	email, username, passwordHash string,
	presence Presence,
	verifiedAt *time.Time,
	createdAt, updatedAt time.Time,
) *User {
	return &User{
		id:           id,
		email:        email,
		username:     username,
		passwordHash: passwordHash,
		presence:     presence,
		verifiedAt:   verifiedAt,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}
}

// --- Domain Behaviors (State Transitions) ---

// Verify marks the user as verified if they aren't already.
func (u *User) Verify(now time.Time) error {
	if u.IsVerified() {
		return ErrAlreadyVerified
	}
	u.verifiedAt = &now
	u.updatedAt = now
	return nil
}

// SetPresence updates the user's presence state safely.
func (u *User) SetPresence(p Presence, now time.Time) error {
	if !p.Valid() {
		return ErrInvalidPresence
	}
	u.presence = p
	u.updatedAt = now
	return nil
}

// UpdatePassword updates the stored password hash.
func (u *User) UpdatePassword(newPasswordHash string, now time.Time) {
	u.passwordHash = newPasswordHash
	u.updatedAt = now
}

// --- Domain Queries / Getters ---

func (u *User) ID() uuid.UUID          { return u.id }
func (u *User) Email() string          { return u.email }
func (u *User) Username() string       { return u.username }
func (u *User) PasswordHash() string   { return u.passwordHash }
func (u *User) Presence() Presence     { return u.presence }
func (u *User) VerifiedAt() *time.Time { return u.verifiedAt }
func (u *User) IsVerified() bool       { return u.verifiedAt != nil }
func (u *User) CreatedAt() time.Time   { return u.createdAt }
func (u *User) UpdatedAt() time.Time   { return u.updatedAt }

// --- UserProfile ---

type UserProfile struct {
	userID      uuid.UUID
	displayName string
	avatarURL   *string
	createdAt   time.Time
	updatedAt   time.Time
}

// NewProfile creates a new profile entity with validation.
func NewProfile(userID uuid.UUID, displayName string, now time.Time) (*UserProfile, error) {
	// Add display name validation here if needed
	return &UserProfile{
		userID:      userID,
		displayName: displayName,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

func ReconstituteProfile(
	userID uuid.UUID,
	displayName string,
	avatarURL *string,
	createdAt, updatedAt time.Time,
) *UserProfile {
	return &UserProfile{
		userID:      userID,
		displayName: displayName,
		avatarURL:   avatarURL,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}

// Domain Behaviors
func (up *UserProfile) UpdateDisplayName(name string, now time.Time) {
	up.displayName = name
	up.updatedAt = now
}

func (up *UserProfile) SetAvatarURL(url *string, now time.Time) {
	up.avatarURL = url
	up.updatedAt = now
}

// Getters
func (up *UserProfile) UserID() uuid.UUID    { return up.userID }
func (up *UserProfile) DisplayName() string  { return up.displayName }
func (up *UserProfile) AvatarURL() *string   { return up.avatarURL }
func (up *UserProfile) CreatedAt() time.Time { return up.createdAt }
func (up *UserProfile) UpdatedAt() time.Time { return up.updatedAt }
