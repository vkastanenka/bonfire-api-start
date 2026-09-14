package user

import (
	"fmt"
	"time"

	"bonfire-api/internal/presence"

	"github.com/google/uuid"
)

const AnonymizeBatchSize = 100
const ScheduleDeleteGracePeriod = 30 * 24 * time.Hour

type User struct {
	ID                     uuid.UUID
	Email                  string
	Username               string
	DisplayName            string
	PasswordHash           string
	Phone                  *string
	Bio                    *string
	AvatarURL              *string
	BannerColor            *string
	PreferredPresence      *presence.Presence
	PreferredPresenceUntil *time.Time
	VerifiedAt             *time.Time
	DisabledAt             *time.Time
	DeleteScheduledAt      *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func Reconstitute(
	id uuid.UUID,
	email string,
	username string,
	passwordHash string,
	phone *string,
	displayName string,
	bio *string,
	avatarURL *string,
	bannerColor *string,
	preferredPresence *presence.Presence,
	preferredPresenceUntil *time.Time,
	verifiedAt *time.Time,
	disabledAt *time.Time,
	deleteScheduledAt *time.Time,
	createdAt time.Time,
	updatedAt time.Time,
) *User {
	return &User{
		ID:                     id,
		Email:                  email,
		Username:               username,
		PasswordHash:           passwordHash,
		Phone:                  phone,
		DisplayName:            displayName,
		Bio:                    bio,
		AvatarURL:              avatarURL,
		BannerColor:            bannerColor,
		PreferredPresence:      preferredPresence,
		PreferredPresenceUntil: preferredPresenceUntil,
		VerifiedAt:             verifiedAt,
		DisabledAt:             disabledAt,
		DeleteScheduledAt:      deleteScheduledAt,
		CreatedAt:              createdAt,
		UpdatedAt:              updatedAt,
	}
}

func New(id uuid.UUID, email, username, displayName, passwordHash string, now time.Time) *User {
	now = now.UTC()
	return &User{
		ID:           id,
		Email:        email,
		Username:     username,
		DisplayName:  displayName,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func (u *User) IsVerified() bool             { return u.VerifiedAt != nil }
func (u *User) IsDisabled() bool             { return u.DisabledAt != nil }
func (u *User) IsScheduledForDeletion() bool { return u.DeleteScheduledAt != nil }

func (u *User) EffectivePresence(now time.Time) *presence.Presence {
	if u.PreferredPresenceUntil != nil && now.After(*u.PreferredPresenceUntil) {
		return nil
	}
	return u.PreferredPresence
}

func (u *User) EnsureActive() error {
	if u.IsDisabled() {
		return ErrUserDisabled
	}
	if u.IsScheduledForDeletion() {
		return ErrUserScheduledDeletion
	}
	return nil
}

func (u *User) Verify(now time.Time) {
	if u.VerifiedAt == nil {
		nowUTC := now.UTC()
		u.VerifiedAt = &nowUTC
		u.touch(nowUTC)
	}
}

func (u *User) Disable(now time.Time) {
	if u.DisabledAt == nil {
		nowUTC := now.UTC()
		u.DisabledAt = &nowUTC
		u.touch(nowUTC)
	}
}

func (u *User) Enable(now time.Time) {
	if u.DisabledAt != nil {
		u.DisabledAt = nil
		u.touch(now)
	}
}

func (u *User) ScheduleDelete(scheduledAt time.Time, now time.Time) {
	if u.DeleteScheduledAt == nil {
		scheduledUTC := scheduledAt.UTC()
		u.DeleteScheduledAt = &scheduledUTC
		u.Disable(now)
	}
}

func (u *User) CancelDelete(now time.Time) {
	if u.DeleteScheduledAt != nil {
		u.DeleteScheduledAt = nil
		u.Enable(now)
	}
}

func (u *User) UpdateEmail(newEmail string, now time.Time) {
	if u.Email != newEmail {
		u.Email = newEmail
		u.touch(now)
	}
}

func (u *User) UpdateUsername(newUsername string, now time.Time) {
	if u.Username != newUsername {
		u.Username = newUsername
		u.touch(now)
	}
}

func (u *User) UpdatePhone(newPhone *string, now time.Time) {
	u.Phone = newPhone
	u.touch(now)
}

func (u *User) UpdatePasswordHash(newHash string, now time.Time) {
	u.PasswordHash = newHash
	u.touch(now)
}

func (u *User) UpdateProfile(displayName string, bio, avatarURL, bannerColor *string, now time.Time) {
	u.DisplayName = displayName
	u.Bio = bio
	u.AvatarURL = avatarURL
	u.BannerColor = bannerColor
	u.touch(now)
}

func (u *User) UpdatePreferredPresence(p *presence.Presence, until *time.Time, now time.Time) {
	u.PreferredPresence = p
	if until != nil {
		t := until.UTC()
		u.PreferredPresenceUntil = &t
	} else {
		u.PreferredPresenceUntil = nil
	}
	u.touch(now)
}

func (u *User) Anonymize(now time.Time) {
	anonID := u.ID.String()
	nowUTC := now.UTC()

	u.Email = fmt.Sprintf("deleted-%s@deleted.invalid", anonID)
	u.Username = fmt.Sprintf("deleted_%s", anonID[:8])
	u.DisplayName = "Deleted User"
	u.PasswordHash = ""
	u.Phone = nil
	u.Bio = nil
	u.AvatarURL = nil
	u.BannerColor = nil
	u.PreferredPresence = nil
	u.PreferredPresenceUntil = nil
	u.VerifiedAt = nil
	u.DeleteScheduledAt = nil
	u.DisabledAt = &nowUTC
	u.touch(nowUTC)
}

func (u *User) touch(at time.Time) {
	u.UpdatedAt = at.UTC()
}
