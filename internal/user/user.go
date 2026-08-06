package user

import (
	"errors"
	"time"

	"bonfire-api/internal/presence"

	"github.com/google/uuid"
)

type User struct {
	id                     ID
	email                  Email
	username               Username
	phone                  Phone
	passwordHash           string
	preferredPresence      *presence.Presence
	preferredPresenceUntil *time.Time
	verifiedAt             *time.Time
	disabledAt             *time.Time
	deleteScheduledAt      *time.Time
	createdAt              time.Time
	updatedAt              time.Time

	profile Profile
}

func (u *User) ID() ID                                { return u.id }
func (u *User) Email() Email                          { return u.email }
func (u *User) Username() Username                    { return u.username }
func (u *User) Phone() Phone                          { return u.phone }
func (u *User) PasswordHash() string                  { return u.passwordHash }
func (u *User) PreferredPresence() *presence.Presence { return u.preferredPresence }
func (u *User) PreferredPresenceUntil() *time.Time    { return u.preferredPresenceUntil }
func (u *User) VerifiedAt() *time.Time                { return u.verifiedAt }
func (u *User) DisabledAt() *time.Time                { return u.disabledAt }
func (u *User) DeleteScheduledAt() *time.Time         { return u.deleteScheduledAt }
func (u *User) CreatedAt() time.Time                  { return u.createdAt }
func (u *User) UpdatedAt() time.Time                  { return u.updatedAt }
func (u *User) Profile() Profile                      { return u.profile }

// --- Profile Value Object ---

type Profile struct {
	displayName ProfileDisplayName
	bio         Bio
	avatarURL   URL
	bannerColor BannerColor
	createdAt   time.Time
	updatedAt   time.Time
}

func (p Profile) DisplayName() ProfileDisplayName { return p.displayName }
func (p Profile) Bio() Bio                        { return p.bio }
func (p Profile) AvatarURL() URL                  { return p.avatarURL }
func (p Profile) BannerColor() BannerColor        { return p.bannerColor }
func (p Profile) CreatedAt() time.Time            { return p.createdAt }
func (p Profile) UpdatedAt() time.Time            { return p.updatedAt }

// --- Redis Cache DTO ---

type CachedUser struct {
	ID                     uuid.UUID `redis:"id"`
	Email                  string    `redis:"email"`
	Username               string    `redis:"username"`
	Phone                  *string   `redis:"phone"`
	DisplayName            string    `redis:"display_name"`
	Bio                    *string   `redis:"bio"`
	AvatarURL              *string   `redis:"avatar_url"`
	BannerColor            *string   `redis:"banner_color"`
	PreferredPresence      int16     `redis:"preferred_presence"`
	PreferredPresenceUntil *int64    `redis:"preferred_presence_until"`
	VerifiedAt             *int64    `redis:"verified_at"`
	DisabledAt             *int64    `redis:"disabled_at"`
	DeleteScheduledAt      *int64    `redis:"delete_scheduled_at"`
	CreatedAt              int64     `redis:"created_at"`
	UpdatedAt              int64     `redis:"updated_at"`
}

// --- Domain Mutators & Behavior ---

func (u *User) UpdateProfile(displayName ProfileDisplayName, bio Bio, avatarURL URL, bannerColor BannerColor) {
	now := time.Now().UTC()
	u.profile.displayName = displayName
	u.profile.bio = bio
	u.profile.avatarURL = avatarURL
	u.profile.bannerColor = bannerColor
	u.profile.updatedAt = now
	u.updatedAt = now
}

func (u *User) Verify(at time.Time) {
	if u.verifiedAt == nil {
		t := at.UTC()
		u.verifiedAt = &t
		u.updatedAt = t
	}
}

func (u *User) UpdateEmail(newEmail Email) {
	if u.email != newEmail {
		u.email = newEmail
		u.updatedAt = time.Now().UTC()
	}
}

func (u *User) UpdatePhone(newPhone Phone) {
	u.phone = newPhone
	u.updatedAt = time.Now().UTC()
}

func (u *User) UpdatePassword(newHash string) error {
	if newHash == "" {
		return errors.New("password hash cannot be empty")
	}
	u.passwordHash = newHash
	u.updatedAt = time.Now().UTC()
	return nil
}

func (u *User) UpdateUsername(newUsername Username) {
	if u.username != newUsername {
		u.username = newUsername
		u.updatedAt = time.Now().UTC()
	}
}

func (u *User) SetPreferredPresence(p *presence.Presence, until *time.Time) error {
	if p != nil && !p.IsValid() {
		return presence.ErrInvalidPresence
	}
	u.preferredPresence = p
	if until != nil {
		t := until.UTC()
		u.preferredPresenceUntil = &t
	} else {
		u.preferredPresenceUntil = nil
	}
	u.updatedAt = time.Now().UTC()
	return nil
}

// --- Constructors & Reconstitution ---

func New(email Email, username Username, passwordHash string, displayName ProfileDisplayName) (*User, error) {
	if passwordHash == "" {
		return nil, errors.New("password hash cannot be empty")
	}

	now := time.Now().UTC()
	id, err := NewID(uuid.Must(uuid.NewV7()))
	if err != nil {
		return nil, err
	}

	return &User{
		id:           id,
		email:        email,
		username:     username,
		passwordHash: passwordHash,
		createdAt:    now,
		updatedAt:    now,
		profile: Profile{
			displayName: displayName,
			createdAt:   now,
			updatedAt:   now,
		},
	}, nil
}

func Reconstitute(
	id ID,
	email Email,
	username Username,
	phone Phone,
	passwordHash string,
	preferredPresence *presence.Presence,
	preferredPresenceUntil *time.Time,
	verifiedAt *time.Time,
	disabledAt *time.Time,
	deleteScheduledAt *time.Time,
	createdAt, updatedAt time.Time,
	profile Profile,
) *User {
	return &User{
		id:                     id,
		email:                  email,
		username:               username,
		phone:                  phone,
		passwordHash:           passwordHash,
		preferredPresence:      preferredPresence,
		preferredPresenceUntil: preferredPresenceUntil,
		verifiedAt:             verifiedAt,
		disabledAt:             disabledAt,
		deleteScheduledAt:      deleteScheduledAt,
		createdAt:              createdAt,
		updatedAt:              updatedAt,
		profile:                profile,
	}
}

func ReconstituteProfile(
	displayName ProfileDisplayName,
	bio Bio,
	avatarURL URL,
	bannerColor BannerColor,
	createdAt, updatedAt time.Time,
) Profile {
	return Profile{
		displayName: displayName,
		bio:         bio,
		avatarURL:   avatarURL,
		bannerColor: bannerColor,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}

// --- Mapping Methods ---

func (u *User) ToCachedUser() *CachedUser {
	var prefPresence int16
	if u.preferredPresence != nil {
		prefPresence = int16(*u.preferredPresence)
	}

	var prefPresenceUntil *int64
	if u.preferredPresenceUntil != nil {
		t := u.preferredPresenceUntil.Unix()
		prefPresenceUntil = &t
	}

	var verifiedAt *int64
	if u.verifiedAt != nil {
		t := u.verifiedAt.Unix()
		verifiedAt = &t
	}

	var disabledAt *int64
	if u.disabledAt != nil {
		t := u.disabledAt.Unix()
		disabledAt = &t
	}

	var deleteScheduledAt *int64
	if u.deleteScheduledAt != nil {
		t := u.deleteScheduledAt.Unix()
		deleteScheduledAt = &t
	}

	return &CachedUser{
		ID:                     u.id.UUID(),
		Email:                  u.email.String(),
		Username:               u.username.String(),
		Phone:                  u.phone.String(),
		DisplayName:            u.profile.displayName.String(),
		Bio:                    u.profile.bio.NilString(),
		AvatarURL:              u.profile.avatarURL.String(),
		BannerColor:            u.profile.bannerColor.String(),
		PreferredPresence:      prefPresence,
		PreferredPresenceUntil: prefPresenceUntil,
		VerifiedAt:             verifiedAt,
		DisabledAt:             disabledAt,
		DeleteScheduledAt:      deleteScheduledAt,
		CreatedAt:              u.createdAt.Unix(),
		UpdatedAt:              u.updatedAt.Unix(),
	}
}

func FromCachedUser(c *CachedUser) (*User, error) {
	if c == nil {
		return nil, nil
	}

	id, err := NewID(c.ID)
	if err != nil {
		return nil, err
	}

	email, err := NewEmail(c.Email)
	if err != nil {
		return nil, err
	}

	username, err := NewUsername(c.Username)
	if err != nil {
		return nil, err
	}

	phone, err := NewPhone(c.Phone)
	if err != nil {
		return nil, err
	}

	displayName, err := NewProfileDisplayName(c.DisplayName)
	if err != nil {
		return nil, err
	}

	var bio Bio
	if c.Bio != nil {
		b, err := NewBio(*c.Bio)
		if err != nil {
			return nil, err
		}
		bio = b
	}

	avatarURL, err := NewURL(c.AvatarURL)
	if err != nil {
		return nil, err
	}

	bannerColor, err := NewBannerColor(c.BannerColor)
	if err != nil {
		return nil, err
	}

	var prefPresence *presence.Presence
	if c.PreferredPresence != 0 {
		p := presence.Presence(c.PreferredPresence)
		prefPresence = &p
	}

	var prefPresenceUntil *time.Time
	if c.PreferredPresenceUntil != nil {
		t := time.Unix(*c.PreferredPresenceUntil, 0).UTC()
		prefPresenceUntil = &t
	}

	var verifiedAt *time.Time
	if c.VerifiedAt != nil {
		t := time.Unix(*c.VerifiedAt, 0).UTC()
		verifiedAt = &t
	}

	var disabledAt *time.Time
	if c.DisabledAt != nil {
		t := time.Unix(*c.DisabledAt, 0).UTC()
		disabledAt = &t
	}

	var deleteScheduledAt *time.Time
	if c.DeleteScheduledAt != nil {
		t := time.Unix(*c.DeleteScheduledAt, 0).UTC()
		deleteScheduledAt = &t
	}

	prof := ReconstituteProfile(
		displayName,
		bio,
		avatarURL,
		bannerColor,
		time.Unix(c.CreatedAt, 0).UTC(),
		time.Unix(c.UpdatedAt, 0).UTC(),
	)

	return Reconstitute(
		id,
		email,
		username,
		phone,
		"", // PasswordHash is intentionally excluded from the cache layer
		prefPresence,
		prefPresenceUntil,
		verifiedAt,
		disabledAt,
		deleteScheduledAt,
		time.Unix(c.CreatedAt, 0).UTC(),
		time.Unix(c.UpdatedAt, 0).UTC(),
		prof,
	), nil
}
