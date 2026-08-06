package user

import (
	"time"

	"bonfire-api/internal/pkg/ptr"

	"github.com/google/uuid"
)

// ============================================================================
// Entities
// ============================================================================

type User struct {
	id                     ID
	email                  Email
	username               Username
	passwordHash           Password
	phone                  *Phone
	preferredPresence      *PreferredPresence
	preferredPresenceUntil *time.Time
	verifiedAt             *time.Time
	disabledAt             *time.Time
	deleteScheduledAt      *time.Time
	createdAt              time.Time
	updatedAt              time.Time
}

func (u *User) ID() ID                                { return u.id }
func (u *User) Email() Email                          { return u.email }
func (u *User) Username() Username                    { return u.username }
func (u *User) PasswordHash() Password                { return u.passwordHash }
func (u *User) Phone() *Phone                         { return u.phone }
func (u *User) PreferredPresence() *PreferredPresence { return u.preferredPresence }
func (u *User) PreferredPresenceUntil() *time.Time    { return u.preferredPresenceUntil }
func (u *User) VerifiedAt() *time.Time                { return u.verifiedAt }
func (u *User) DisabledAt() *time.Time                { return u.disabledAt }
func (u *User) DeleteScheduledAt() *time.Time         { return u.deleteScheduledAt }
func (u *User) CreatedAt() time.Time                  { return u.createdAt }
func (u *User) UpdatedAt() time.Time                  { return u.updatedAt }

type Profile struct {
	userID      ID
	displayName DisplayName
	bio         *Bio
	avatarURL   *URL
	bannerColor *BannerColor
	updatedAt   time.Time
}

func (p *Profile) UserID() ID                { return p.userID }
func (p *Profile) DisplayName() DisplayName  { return p.displayName }
func (p *Profile) Bio() *Bio                 { return p.bio }
func (p *Profile) AvatarURL() *URL           { return p.avatarURL }
func (p *Profile) BannerColor() *BannerColor { return p.bannerColor }
func (p *Profile) UpdatedAt() time.Time      { return p.updatedAt }

type Aggregate struct {
	user    *User
	profile *Profile
}

func NewAggregate(u *User, p *Profile) *Aggregate {
	return &Aggregate{
		user:    u,
		profile: p,
	}
}

func (ua *Aggregate) User() *User       { return ua.user }
func (ua *Aggregate) Profile() *Profile { return ua.profile }

// ============================================================================
// User Methods
// ============================================================================

func New(email Email, username Username, passwordHash Password) (*User, error) {
	now := time.Now().UTC()

	v7, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	id, err := NewID(v7)
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
	}, nil
}

func Reconstitute(
	id ID,
	email Email,
	username Username,
	phone *Phone,
	passwordHash Password,
	preferredPresence *PreferredPresence,
	preferredPresenceUntil *time.Time,
	verifiedAt *time.Time,
	disabledAt *time.Time,
	deleteScheduledAt *time.Time,
	createdAt, updatedAt time.Time,
) *User {
	return &User{
		id:                     id,
		email:                  email,
		username:               username,
		phone:                  phone,
		passwordHash:           passwordHash,
		preferredPresence:      preferredPresence,
		preferredPresenceUntil: ptr.Map(preferredPresenceUntil, time.Time.UTC),
		verifiedAt:             ptr.Map(verifiedAt, time.Time.UTC),
		disabledAt:             ptr.Map(disabledAt, time.Time.UTC),
		deleteScheduledAt:      ptr.Map(deleteScheduledAt, time.Time.UTC),
		createdAt:              createdAt.UTC(),
		updatedAt:              updatedAt.UTC(),
	}
}

func (u *User) Verify(at time.Time) {
	if u.verifiedAt == nil {
		u.verifiedAt = ptr.To(at.UTC())
		u.touchAt(at)
	}
}

func (u *User) Disable(at time.Time) {
	if u.disabledAt == nil {
		u.disabledAt = ptr.To(at.UTC())
		u.touchAt(at)
	}
}

func (u *User) Enable() {
	if u.disabledAt != nil {
		u.disabledAt = nil
		u.touch()
	}
}

func (u *User) ScheduleDeletion(at time.Time) {
	u.deleteScheduledAt = ptr.To(at.UTC())
	u.touchAt(at)
}

func (u *User) CancelDeletion() {
	if u.deleteScheduledAt != nil {
		u.deleteScheduledAt = nil
		u.touch()
	}
}

func (u *User) UpdateEmail(newEmail Email) {
	if !u.email.Equals(newEmail) {
		u.email = newEmail
		u.touch()
	}
}

func (u *User) UpdateUsername(newUsername Username) {
	if !u.username.Equals(newUsername) {
		u.username = newUsername
		u.touch()
	}
}

func (u *User) UpdatePhone(newPhone *Phone) {
	u.phone = newPhone
	u.touch()
}

func (u *User) UpdatePassword(newHash Password) {
	u.passwordHash = newHash
	u.touch()
}

func (u *User) SetPreferredPresence(p *PreferredPresence, until *time.Time) {
	u.preferredPresence = p
	u.preferredPresenceUntil = ptr.Map(until, time.Time.UTC)
	u.touch()
}

func (u *User) touch() {
	u.updatedAt = time.Now().UTC()
}

func (u *User) touchAt(at time.Time) {
	u.updatedAt = at.UTC()
}

// ============================================================================
// Profile Methods
// ============================================================================

func NewProfile(userID ID, displayName DisplayName) *Profile {
	now := time.Now().UTC()
	return &Profile{
		userID:      userID,
		displayName: displayName,
		updatedAt:   now,
	}
}

func ReconstituteProfile(
	userID ID,
	displayName DisplayName,
	bio *Bio,
	avatarURL *URL,
	bannerColor *BannerColor,
	updatedAt time.Time,
) *Profile {
	return &Profile{
		userID:      userID,
		displayName: displayName,
		bio:         bio,
		avatarURL:   avatarURL,
		bannerColor: bannerColor,
		updatedAt:   updatedAt.UTC(),
	}
}

func (p *Profile) Update(displayName DisplayName, bio *Bio, avatarURL *URL, bannerColor *BannerColor) {
	p.displayName = displayName
	p.bio = bio
	p.avatarURL = avatarURL
	p.bannerColor = bannerColor
	p.updatedAt = time.Now().UTC()
}

// ============================================================================
// Cache Mapping
// ============================================================================

type CachedAggregate struct {
	ID                     uuid.UUID `redis:"id"`
	Email                  string    `redis:"email"`
	Username               string    `redis:"username"`
	Phone                  *string   `redis:"phone"`
	DisplayName            string    `redis:"display_name"`
	Bio                    *string   `redis:"bio"`
	AvatarURL              *string   `redis:"avatar_url"`
	BannerColor            *string   `redis:"banner_color"`
	PreferredPresence      *string   `redis:"preferred_presence"`
	PreferredPresenceUntil *int64    `redis:"preferred_presence_until"`
	VerifiedAt             *int64    `redis:"verified_at"`
	DisabledAt             *int64    `redis:"disabled_at"`
	DeleteScheduledAt      *int64    `redis:"delete_scheduled_at"`
	CreatedAt              int64     `redis:"created_at"`
	UpdatedAt              int64     `redis:"updated_at"`
	ProfileUpdatedAt       int64     `redis:"profile_updated_at"`
}

func (ua Aggregate) ToCachedAggregate() *CachedAggregate {
	u := ua.user
	p := ua.profile

	var phone *string
	if u.phone != nil {
		s := u.phone.String()
		phone = &s
	}

	var bio *string
	if p.bio != nil {
		s := p.bio.String()
		bio = &s
	}

	var avatarURL *string
	if p.avatarURL != nil {
		s := p.avatarURL.String()
		avatarURL = &s
	}

	var bannerColor *string
	if p.bannerColor != nil {
		s := p.bannerColor.String()
		bannerColor = &s
	}

	var prefPresence *string
	if u.preferredPresence != nil {
		s := u.preferredPresence.String()
		prefPresence = &s
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

	return &CachedAggregate{
		ID:                     u.id.UUID(),
		Email:                  u.email.String(),
		Username:               u.username.String(),
		Phone:                  phone,
		DisplayName:            p.displayName.String(),
		Bio:                    bio,
		AvatarURL:              avatarURL,
		BannerColor:            bannerColor,
		PreferredPresence:      prefPresence,
		PreferredPresenceUntil: prefPresenceUntil,
		VerifiedAt:             verifiedAt,
		DisabledAt:             disabledAt,
		DeleteScheduledAt:      deleteScheduledAt,
		CreatedAt:              u.createdAt.Unix(),
		UpdatedAt:              u.updatedAt.Unix(),
		ProfileUpdatedAt:       p.updatedAt.Unix(),
	}
}

func FromCachedAggregate(c *CachedAggregate) (*Aggregate, error) {
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

	var phone *Phone
	if c.Phone != nil {
		p, err := NewPhone(c.Phone)
		if err != nil {
			return nil, err
		}
		phone = &p
	}

	displayName, err := NewDisplayName(c.DisplayName)
	if err != nil {
		return nil, err
	}

	var bio *Bio
	if c.Bio != nil {
		b, err := NewBio(c.Bio)
		if err != nil {
			return nil, err
		}
		bio = &b
	}

	var avatarURL *URL
	if c.AvatarURL != nil {
		u, err := NewURL(c.AvatarURL)
		if err != nil {
			return nil, err
		}
		avatarURL = &u
	}

	var bannerColor *BannerColor
	if c.BannerColor != nil {
		bc, err := NewBannerColor(c.BannerColor)
		if err != nil {
			return nil, err
		}
		bannerColor = &bc
	}

	var prefPresence *PreferredPresence
	if c.PreferredPresence != nil {
		pp, err := NewPreferredPresence(c.PreferredPresence)
		if err != nil {
			return nil, err
		}
		prefPresence = &pp
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

	u := Reconstitute(
		id,
		email,
		username,
		phone,
		Password{}, // Hash intentionally omitted from Redis cache
		prefPresence,
		prefPresenceUntil,
		verifiedAt,
		disabledAt,
		deleteScheduledAt,
		time.Unix(c.CreatedAt, 0).UTC(),
		time.Unix(c.UpdatedAt, 0).UTC(),
	)

	p := ReconstituteProfile(
		id,
		displayName,
		bio,
		avatarURL,
		bannerColor,
		time.Unix(c.ProfileUpdatedAt, 0).UTC(),
	)

	agg := NewAggregate(*u, p)
	return &agg, nil
}
