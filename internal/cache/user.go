package cache

import (
	"context"
	"fmt"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

func UserKey(id fields.ID) string {
	return fmt.Sprintf("users:%s", id.String())
}

type User struct {
	ID                     uuid.UUID `json:"id"`
	Email                  string    `json:"email"`
	Username               string    `json:"username"`
	DisplayName            string    `json:"display_name"`
	Phone                  string    `json:"phone"`
	Bio                    string    `json:"bio"`
	AvatarURL              string    `json:"avatar_url"`
	BannerColor            string    `json:"banner_color"`
	PreferredPresence      int16     `json:"preferred_presence"`
	PreferredPresenceUntil int64     `json:"preferred_presence_until"`
	VerifiedAt             int64     `json:"verified_at"`
	DisabledAt             int64     `json:"disabled_at"`
	DeleteScheduledAt      int64     `json:"delete_scheduled_at"`
	CreatedAt              int64     `json:"created_at"`
	UpdatedAt              int64     `json:"updated_at"`
}

func (u User) ToDomain() (*user.User, error) {
	id, err := fields.ParseRequiredID("id", u.ID)
	if err != nil {
		return nil, err
	}
	email, err := user.ParseEmail("email", u.Email)
	if err != nil {
		return nil, err
	}
	username, err := user.ParseUsername("username", u.Username)
	if err != nil {
		return nil, err
	}
	displayName, err := user.ParseDisplayName("display_name", u.DisplayName)
	if err != nil {
		return nil, err
	}
	phone, err := user.ParsePhone("phone", u.Phone)
	if err != nil {
		return nil, err
	}
	bio, err := user.ParseBio("bio", u.Bio)
	if err != nil {
		return nil, err
	}
	avatarURL, err := fields.ParseURL("avatar_url", u.AvatarURL)
	if err != nil {
		return nil, err
	}
	bannerColor, err := fields.ParseHexColor("banner_color", u.BannerColor)
	if err != nil {
		return nil, err
	}
	prefPresence, err := user.ParsePreferredPresenceFromInt16("preferred_presence", u.PreferredPresence)
	if err != nil {
		return nil, err
	}

	return user.Reconstitute(
		id,
		email,
		username,
		user.PasswordHash{},
		phone,
		displayName,
		bio,
		avatarURL,
		bannerColor,
		prefPresence,
		fields.NewTimestampFromUnix(u.PreferredPresenceUntil),
		fields.NewTimestampFromUnix(u.VerifiedAt),
		fields.NewTimestampFromUnix(u.DisabledAt),
		fields.NewTimestampFromUnix(u.DeleteScheduledAt),
		fields.NewTimestampFromUnix(u.CreatedAt),
		fields.NewTimestampFromUnix(u.UpdatedAt),
	), nil
}

func FromDomain(u *user.User) *User {
	return &User{
		ID:                     u.ID().UUID(),
		Email:                  u.Email().String(),
		Username:               u.Username().String(),
		DisplayName:            u.DisplayName().String(),
		Phone:                  u.Phone().String(),
		Bio:                    u.Bio().String(),
		AvatarURL:              u.AvatarURL().String(),
		BannerColor:            u.BannerColor().String(),
		PreferredPresence:      u.PreferredPresence().Int16(),
		PreferredPresenceUntil: u.PreferredPresenceUntil().Unix(),
		VerifiedAt:             u.VerifiedAt().Unix(),
		DisabledAt:             u.DisabledAt().Unix(),
		DeleteScheduledAt:      u.DeleteScheduledAt().Unix(),
		CreatedAt:              u.CreatedAt().Unix(),
		UpdatedAt:              u.UpdatedAt().Unix(),
	}
}

type UserCache struct {
	engine *KeyCache[fields.ID, User]
}

func NewUserCache(store *redis.Store, ttl time.Duration) *UserCache {
	engine := NewKeyCache[fields.ID, User](
		store.WithScope(redis.ScopeUser),
		ttl,
		UserKey,
	)
	return &UserCache{engine: engine}
}

func (u *UserCache) Set(ctx context.Context, usr *user.User) error {
	if usr == nil {
		return nil
	}
	return u.engine.Set(ctx, usr.ID(), FromDomain(usr))
}

func (u *UserCache) SetBatch(ctx context.Context, users []*user.User) error {
	if len(users) == 0 {
		return nil
	}
	items := make(map[fields.ID]*User, len(users))
	for _, usr := range users {
		if usr != nil {
			items[usr.ID()] = FromDomain(usr)
		}
	}
	return u.engine.SetBatch(ctx, items)
}

func (u *UserCache) Get(ctx context.Context, userID fields.ID) (*user.User, error) {
	if !userID.IsValid() {
		return nil, nil
	}
	cached, err := u.engine.Get(ctx, userID)
	if err != nil || cached == nil {
		return nil, err
	}
	return cached.ToDomain()
}

func (u *UserCache) GetBatch(ctx context.Context, userIDs []fields.ID) (map[fields.ID]*user.User, []fields.ID, error) {
	found, missing, err := u.engine.GetBatch(ctx, userIDs)
	if err != nil {
		return nil, nil, err
	}

	result := make(map[fields.ID]*user.User, len(found))
	for id, cached := range found {
		domainUser, err := cached.ToDomain()
		if err != nil {
			missing = append(missing, id)
			continue
		}
		result[id] = domainUser
	}

	return result, missing, nil
}

func (u *UserCache) Delete(ctx context.Context, userID fields.ID) error {
	if !userID.IsValid() {
		return nil
	}
	return u.engine.Delete(ctx, userID)
}

func (u *UserCache) DeleteBatch(ctx context.Context, userIDs []fields.ID) error {
	return u.engine.DeleteBatch(ctx, userIDs)
}

// PublishEvent broadcasts a user domain event over the user's WebSocket channel.
// func (u *UserCache) PublishEvent(ctx context.Context, userID string, eventType string, payload any) error {
// 	channel := fmt.Sprintf("user:%s:events", userID)
// 	wsEvent := map[string]any{
// 		"type": eventType,
// 		"data": payload,
// 	}
// 	return u.engine.Store().Publish(ctx, channel, wsEvent)
// }

// InvalidateSession clears active user session keys from Redis. (TODO: Move to session cache)
// func (u *UserCache) InvalidateSession(ctx context.Context, userID string) error {
// 	sessionKey := fmt.Sprintf("user:%s:session", userID)
// 	return u.engine.Store().Delete(ctx, sessionKey)
// }
