package user

import (
	"bonfire-api/internal/presence"
	"time"

	"github.com/google/uuid"
)

type User struct {
	id                uuid.UUID
	email             Email
	username          Username
	passwordHash      string
	preferredPresence *presence.Presence
	verifiedAt        *time.Time
	createdAt         time.Time
	updatedAt         time.Time

	profile Profile
}

func (u *User) UpdateProfile(displayName ProfileDisplayName, avatarURL *string) {
	now := time.Now().UTC()
	u.profile.displayName = displayName
	u.profile.avatarURL = avatarURL
	u.profile.updatedAt = now
	u.updatedAt = now
}

func (u *User) Verify(at time.Time) {
	if u.verifiedAt == nil {
		u.verifiedAt = &at
		u.updatedAt = time.Now().UTC()
	}
}

func (u *User) ChangePassword(newHash string) {
	u.passwordHash = newHash
	u.updatedAt = time.Now().UTC()
}

func (u *User) SetPreferredPresence(p *presence.Presence) error {
	if p != nil && !p.IsValid() {
		return presence.ErrInvalidPresence
	}

	u.preferredPresence = p
	u.updatedAt = time.Now().UTC()
	return nil
}

func (u *User) ID() uuid.UUID                         { return u.id }
func (u *User) Email() Email                          { return u.email }
func (u *User) Username() Username                    { return u.username }
func (u *User) PasswordHash() string                  { return u.passwordHash }
func (u *User) PreferredPresence() *presence.Presence { return u.preferredPresence }
func (u *User) VerifiedAt() *time.Time                { return u.verifiedAt }
func (u *User) IsVerified() bool                      { return u.verifiedAt != nil }
func (u *User) CreatedAt() time.Time                  { return u.createdAt }
func (u *User) UpdatedAt() time.Time                  { return u.updatedAt }
func (u *User) Profile() Profile                      { return u.profile }

func New(email Email, username Username, passwordHash string, displayName ProfileDisplayName) (*User, error) {
	now := time.Now().UTC()
	userID := uuid.Must(uuid.NewV7())

	return &User{
		id:                userID,
		email:             email,
		username:          username,
		passwordHash:      passwordHash,
		preferredPresence: nil,
		createdAt:         now,
		updatedAt:         now,
		profile: Profile{
			displayName: displayName,
			createdAt:   now,
			updatedAt:   now,
		},
	}, nil
}

func Reconstitute(
	id uuid.UUID,
	email Email,
	username Username,
	passwordHash string,
	preferredPresence *presence.Presence,
	verifiedAt *time.Time,
	createdAt, updatedAt time.Time,
	profile Profile,
) *User {
	return &User{
		id:                id,
		email:             email,
		username:          username,
		passwordHash:      passwordHash,
		preferredPresence: preferredPresence,
		verifiedAt:        verifiedAt,
		createdAt:         createdAt,
		updatedAt:         updatedAt,
		profile:           profile,
	}
}

type Profile struct {
	displayName ProfileDisplayName
	avatarURL   *string
	createdAt   time.Time
	updatedAt   time.Time
}

func ReconstituteProfile(
	displayName ProfileDisplayName,
	avatarURL *string,
	createdAt, updatedAt time.Time,
) Profile {
	return Profile{
		displayName: displayName,
		avatarURL:   avatarURL,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}

func (p Profile) DisplayName() ProfileDisplayName { return p.displayName }
func (p Profile) AvatarURL() *string              { return p.avatarURL }
func (p Profile) CreatedAt() time.Time            { return p.createdAt }
func (p Profile) UpdatedAt() time.Time            { return p.updatedAt }
