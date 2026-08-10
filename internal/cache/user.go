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
)

func userKey(id fields.ID) string {
	return fmt.Sprintf("users:%s", id.String())
}

type UserCache struct {
	store *redis.Store
	ttl   time.Duration
}

func NewUserCache(store *redis.Store, ttl time.Duration) *UserCache {
	return &UserCache{
		store: store.WithScope(redis.ScopeUsers),
		ttl:   ttl,
	}
}

type CachedUser struct {
	ID                     uuid.UUID `redis:"id"`
	Email                  string    `redis:"email"`
	Username               string    `redis:"username"`
	DisplayName            string    `redis:"display_name"`
	Phone                  string    `redis:"phone"`
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

	var preferredPresence int16
	if u.PreferredPresence().IsValid() {
		preferredPresence = u.PreferredPresence().Int16()
	}

	return &CachedUser{
		ID:                     u.ID().UUID(),
		Email:                  u.Email().String(),
		Username:               u.Username().String(),
		DisplayName:            u.DisplayName().String(),
		Phone:                  u.Phone().String(),
		Bio:                    u.Bio().String(),
		AvatarURL:              u.AvatarURL().String(),
		BannerColor:            u.BannerColor().String(),
		PreferredPresence:      preferredPresence,
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

	email, err := user.NewEmail(cu.Email)
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

	phone, err := user.NewPhone(cu.Phone)
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

	var prefPresence user.PreferredPresence
	if cu.PreferredPresence != 0 {
		var err error
		prefPresence, err = user.PreferredPresenceFromInt16(cu.PreferredPresence)
		if err != nil {
			return nil, err
		}
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
		fields.NewTimestampFromUnix(cu.PreferredPresenceUntil),
		fields.NewTimestampFromUnix(cu.VerifiedAt),
		fields.NewTimestampFromUnix(cu.DisabledAt),
		fields.NewTimestampFromUnix(cu.DeleteScheduledAt),
		fields.NewTimestampFromUnix(cu.CreatedAt),
		fields.NewTimestampFromUnix(cu.UpdatedAt),
	), nil
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

	err := u.store.ExecPipeline(ctx, func(pipeCtx context.Context) error {
		for _, cu := range cus {
			key := userKey(fields.ID(cu.ID))
			f := buildCachedUserFields(cu)

			if err := u.store.HMSet(pipeCtx, key, f); err != nil {
				return err
			}
			if err := u.store.Expire(pipeCtx, key, u.ttl); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return u.store.Err(err)
	}

	return nil
}

func (u *UserCache) Get(ctx context.Context, userID fields.ID) (*user.User, error) {
	found, _, err := u.GetBatch(ctx, []fields.ID{userID})
	if err != nil {
		return nil, err
	}

	return found[userID], nil
}

func (u *UserCache) GetBatch(ctx context.Context, userIDs []fields.ID) (map[fields.ID]*user.User, []fields.ID, error) {
	if len(userIDs) == 0 {
		return make(map[fields.ID]*user.User), nil, nil
	}

	uniqueIDs := make([]fields.ID, 0, len(userIDs))
	seen := make(map[fields.ID]struct{}, len(userIDs))
	for _, id := range userIDs {
		if !id.IsValid() {
			continue
		}
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			uniqueIDs = append(uniqueIDs, id)
		}
	}

	if len(uniqueIDs) == 0 {
		return make(map[fields.ID]*user.User), nil, nil
	}

	results := make(map[fields.ID]*map[string]string, len(uniqueIDs))
	for _, id := range uniqueIDs {
		res := make(map[string]string)
		results[id] = &res
	}

	err := u.store.ExecPipeline(ctx, func(pipeCtx context.Context) error {
		for _, id := range uniqueIDs {
			key := userKey(id)
			_ = u.store.HGetAll(pipeCtx, key, results[id])
		}
		return nil
	})
	if err != nil {
		return nil, nil, u.store.Err(err)
	}

	found := make(map[fields.ID]*user.User, len(uniqueIDs))
	var missing []fields.ID

	for id, resPtr := range results {
		res := *resPtr
		if len(res) == 0 {
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

func (u *UserCache) Delete(ctx context.Context, userID fields.ID) error {
	if !userID.IsValid() {
		return nil
	}
	return u.DeleteBatch(ctx, []fields.ID{userID})
}

func (u *UserCache) DeleteBatch(ctx context.Context, userIDs []fields.ID) error {
	if len(userIDs) == 0 {
		return nil
	}

	uniqueIDs := make([]fields.ID, 0, len(userIDs))
	seen := make(map[fields.ID]struct{}, len(userIDs))
	for _, id := range userIDs {
		if !id.IsValid() {
			continue
		}
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			uniqueIDs = append(uniqueIDs, id)
		}
	}

	if len(uniqueIDs) == 0 {
		return nil
	}

	keys := make([]string, 0, len(uniqueIDs))
	for _, id := range uniqueIDs {
		keys = append(keys, userKey(id))
	}

	if err := u.store.Delete(ctx, keys...); err != nil {
		return u.store.Err(err)
	}

	return nil
}

func buildCachedUserFields(cu *CachedUser) map[string]interface{} {
	return map[string]interface{}{
		"id":                       cu.ID.String(),
		"email":                    cu.Email,
		"username":                 cu.Username,
		"display_name":             cu.DisplayName,
		"phone":                    cu.Phone,
		"bio":                      cu.Bio,
		"avatar_url":               cu.AvatarURL,
		"banner_color":             cu.BannerColor,
		"preferred_presence":       strconv.Itoa(int(cu.PreferredPresence)),
		"preferred_presence_until": strconv.FormatInt(cu.PreferredPresenceUntil, 10),
		"verified_at":              strconv.FormatInt(cu.VerifiedAt, 10),
		"disabled_at":              strconv.FormatInt(cu.DisabledAt, 10),
		"delete_scheduled_at":      strconv.FormatInt(cu.DeleteScheduledAt, 10),
		"created_at":               strconv.FormatInt(cu.CreatedAt, 10),
		"updated_at":               strconv.FormatInt(cu.UpdatedAt, 10),
	}
}

func parseCachedUser(m map[string]string) (*CachedUser, error) {
	id, err := uuid.Parse(m["id"])
	if err != nil {
		return nil, err
	}

	prefPresence, _ := strconv.ParseInt(m["preferred_presence"], 10, 16)
	prefPresenceUntil, _ := strconv.ParseInt(m["preferred_presence_until"], 10, 64)
	verifiedAt, _ := strconv.ParseInt(m["verified_at"], 10, 64)
	disabledAt, _ := strconv.ParseInt(m["disabled_at"], 10, 64)
	deleteScheduledAt, _ := strconv.ParseInt(m["delete_scheduled_at"], 10, 64)
	createdAt, _ := strconv.ParseInt(m["created_at"], 10, 64)
	updatedAt, _ := strconv.ParseInt(m["updated_at"], 10, 64)

	return &CachedUser{
		ID:                     id,
		Email:                  m["email"],
		Username:               m["username"],
		DisplayName:            m["display_name"],
		Phone:                  m["phone"],
		Bio:                    m["bio"],
		AvatarURL:              m["avatar_url"],
		BannerColor:            m["banner_color"],
		PreferredPresence:      int16(prefPresence),
		PreferredPresenceUntil: prefPresenceUntil,
		VerifiedAt:             verifiedAt,
		DisabledAt:             disabledAt,
		DeleteScheduledAt:      deleteScheduledAt,
		CreatedAt:              createdAt,
		UpdatedAt:              updatedAt,
	}, nil
}
