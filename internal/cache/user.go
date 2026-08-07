package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type UserCache struct {
	store redis.Store
	ttl   time.Duration
}

func NewUserCache(store redis.Store, ttl time.Duration) *UserCache {
	return &UserCache{
		store: store,
		ttl:   ttl,
	}
}

type CachedUser struct {
	ID                     uuid.UUID `redis:"id"`
	Username               string    `redis:"username"`
	DisplayName            string    `redis:"display_name"`
	Bio                    string    `redis:"bio"`
	AvatarURL              string    `redis:"avatar_url"`
	BannerColor            string    `redis:"banner_color"`
	PreferredPresence      int16     `redis:"preferred_presence"`
	PreferredPresenceUntil int64     `redis:"preferred_presence_until"`
	VerifiedAt             int64     `redis:"verified_at"`
	DisabledAt             int64     `redis:"disabled_at"`
	DeleteScheduledAt      int64     `redis:"delete_scheduled_at"`
	CreatedAt              int64     `redis:"created_at"`
	UpdatedAt              int64     `redis:"updated_at"`
}

func NewCachedUser(u *user.User) *CachedUser {
	if u == nil {
		return nil
	}

	return &CachedUser{
		ID:                     u.ID().UUID(),
		Username:               u.Username().String(),
		DisplayName:            u.DisplayName().String(),
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

func (cu *CachedUser) Reconstitute() (*user.User, error) {
	if cu == nil {
		return nil, nil
	}

	id, err := fields.NewID(cu.ID)
	if err != nil {
		return nil, err
	}

	username, err := user.NewUsername(cu.Username)
	if err != nil {
		return nil, err
	}

	displayName, err := user.NewDisplayName(cu.DisplayName)
	if err != nil {
		return nil, err
	}

	bio, err := user.NewBio(cu.Bio)
	if err != nil {
		return nil, err
	}

	avatarURL, err := fields.NewURL(cu.AvatarURL)
	if err != nil {
		return nil, err
	}

	bannerColor, err := fields.NewHexColor(cu.BannerColor)
	if err != nil {
		return nil, err
	}

	prefPresence, err := user.PreferredPresenceFromInt16(cu.PreferredPresence)
	if err != nil {
		return nil, err
	}

	return user.Reconstitute(
		id,
		user.Email{},
		username,
		user.PasswordHash{},
		user.Phone{},
		displayName,
		bio,
		avatarURL,
		bannerColor,
		prefPresence,
		fields.NewTimestampFromUnix(cu.PreferredPresenceUntil),
		fields.NewTimestampFromUnix(cu.VerifiedAt),
		fields.NewTimestampFromUnix(cu.DisabledAt),
		fields.NewTimestampFromUnix(cu.DeleteScheduledAt),
		fields.NewTimestampFromUnix(cu.CreatedAt),
		fields.NewTimestampFromUnix(cu.UpdatedAt),
	), nil
}

func userCacheKey(userID uuid.UUID) string {
	return fmt.Sprintf("user:%s:profile", userID.String())
}

func (u *UserCache) Set(ctx context.Context, usr *user.User) error {
	if usr == nil {
		return nil
	}
	return u.SetBatch(ctx, []*user.User{usr})
}

func (u *UserCache) SetBatch(ctx context.Context, users []*user.User) error {
	if len(users) == 0 {
		return nil
	}

	cus := make([]*CachedUser, 0, len(users))
	for _, usr := range users {
		if cu := NewCachedUser(usr); cu != nil {
			cus = append(cus, cu)
		}
	}

	if len(cus) == 0 {
		return nil
	}

	err := u.store.ExecPipelineFunc(ctx, func(pipe goredis.Pipeliner) error {
		for _, cu := range cus {
			key := userCacheKey(cu.ID)
			f := buildCachedUserFields(cu)

			pipe.HSet(ctx, key, f)
			pipe.Expire(ctx, key, u.ttl)
		}
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}

	return nil
}

func (u *UserCache) Get(ctx context.Context, userID uuid.UUID) (*user.User, error) {
	found, _, err := u.GetBatch(ctx, []uuid.UUID{userID})
	if err != nil {
		return nil, err
	}

	return found[userID], nil
}

func (u *UserCache) GetBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*user.User, []uuid.UUID, error) {
	if len(userIDs) == 0 {
		return make(map[uuid.UUID]*user.User), nil, nil
	}

	uniqueIDs := make([]uuid.UUID, 0, len(userIDs))
	seen := make(map[uuid.UUID]struct{}, len(userIDs))
	for _, id := range userIDs {
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			uniqueIDs = append(uniqueIDs, id)
		}
	}

	cmds := make(map[uuid.UUID]*goredis.MapStringStringCmd, len(uniqueIDs))
	err := u.store.ExecPipelineFunc(ctx, func(pipe goredis.Pipeliner) error {
		for _, id := range uniqueIDs {
			key := userCacheKey(id)
			cmds[id] = pipe.HGetAll(ctx, key)
		}
		return nil
	})
	if err != nil {
		return nil, nil, redis.NewError(err, redis.ScopeUser)
	}

	found := make(map[uuid.UUID]*user.User, len(uniqueIDs))
	var missing []uuid.UUID

	for id, cmd := range cmds {
		res, resErr := cmd.Result()
		if resErr != nil || len(res) == 0 {
			missing = append(missing, id)
			continue
		}

		cu, parseErr := parseCachedUser(res)
		if parseErr != nil {
			missing = append(missing, id)
			continue
		}

		usr, domainErr := cu.Reconstitute()
		if domainErr != nil || usr == nil {
			missing = append(missing, id)
			continue
		}

		found[id] = usr
	}

	return found, missing, nil
}

func (u *UserCache) Invalidate(ctx context.Context, userID uuid.UUID) error {
	key := userCacheKey(userID)
	if err := u.store.Del(ctx, key); err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}
	return nil
}

func buildCachedUserFields(cu *CachedUser) map[string]interface{} {
	f := map[string]interface{}{
		"id":                 cu.ID.String(),
		"username":           cu.Username,
		"display_name":       cu.DisplayName,
		"bio":                cu.Bio,
		"avatar_url":         cu.AvatarURL,
		"banner_color":       cu.BannerColor,
		"preferred_presence": strconv.Itoa(int(cu.PreferredPresence)),
		"created_at":         strconv.FormatInt(cu.CreatedAt, 10),
		"updated_at":         strconv.FormatInt(cu.UpdatedAt, 10),
	}

	if cu.PreferredPresenceUntil != nil {
		f["preferred_presence_until"] = strconv.FormatInt(*cu.PreferredPresenceUntil, 10)
	}
	if cu.VerifiedAt != nil {
		f["verified_at"] = strconv.FormatInt(*cu.VerifiedAt, 10)
	}
	if cu.DisabledAt != nil {
		f["disabled_at"] = strconv.FormatInt(*cu.DisabledAt, 10)
	}
	if cu.DeleteScheduledAt != nil {
		f["delete_scheduled_at"] = strconv.FormatInt(*cu.DeleteScheduledAt, 10)
	}

	return f
}

func parseCachedUser(m map[string]string) (*CachedUser, error) {
	id, err := uuid.Parse(m["id"])
	if err != nil {
		return nil, err
	}

	prefPresenceInt, _ := strconv.ParseInt(m["preferred_presence"], 10, 16)
	createdAt, _ := strconv.ParseInt(m["created_at"], 10, 64)
	updatedAt, _ := strconv.ParseInt(m["updated_at"], 10, 64)

	cu := &CachedUser{
		ID:                id,
		Username:          m["username"],
		DisplayName:       m["display_name"],
		Bio:               m["bio"],
		AvatarURL:         m["avatar_url"],
		BannerColor:       m["banner_color"],
		PreferredPresence: int16(prefPresenceInt),
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}

	if val, ok := m["preferred_presence_until"]; ok && val != "" {
		if ts, pErr := strconv.ParseInt(val, 10, 64); pErr == nil {
			cu.PreferredPresenceUntil = &ts
		}
	}
	if val, ok := m["verified_at"]; ok && val != "" {
		if ts, pErr := strconv.ParseInt(val, 10, 64); pErr == nil {
			cu.VerifiedAt = &ts
		}
	}
	if val, ok := m["disabled_at"]; ok && val != "" {
		if ts, pErr := strconv.ParseInt(val, 10, 64); pErr == nil {
			cu.DisabledAt = &ts
		}
	}
	if val, ok := m["delete_scheduled_at"]; ok && val != "" {
		if ts, pErr := strconv.ParseInt(val, 10, 64); pErr == nil {
			cu.DeleteScheduledAt = &ts
		}
	}

	return cu, nil
}
