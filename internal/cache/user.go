package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

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

type UserAggregate struct {
	ID                     uuid.UUID `redis:"id"`
	Username               string    `redis:"username"`
	DisplayName            string    `redis:"display_name"`
	Bio                    string    `redis:"bio"`
	AvatarURL              string    `redis:"avatar_url"`
	BannerColor            string    `redis:"banner_color"`
	PreferredPresence      int16     `redis:"preferred_presence"`
	PreferredPresenceUntil *int64    `redis:"preferred_presence_until"`
	VerifiedAt             *int64    `redis:"verified_at"`
	DisabledAt             *int64    `redis:"disabled_at"`
	DeleteScheduledAt      *int64    `redis:"delete_scheduled_at"`
	CreatedAt              int64     `redis:"created_at"`
	UpdatedAt              int64     `redis:"updated_at"`
}

func NewUserAggregate(agg *user.Aggregate) *UserAggregate {
	if agg == nil || agg.User() == nil || agg.Profile() == nil {
		return nil
	}

	u := agg.User()
	p := agg.Profile()

	var bio string
	if p.Bio() != nil {
		bio = p.Bio().String()
	}

	var avatarURL string
	if p.AvatarURL() != nil {
		avatarURL = p.AvatarURL().String()
	}

	var bannerColor string
	if p.BannerColor() != nil {
		bannerColor = p.BannerColor().String()
	}

	var prefPresence int16
	if u.PreferredPresence() != nil {
		prefPresence = u.PreferredPresence().Int16()
	}

	var prefPresenceUntil *int64
	if u.PreferredPresenceUntil() != nil {
		t := u.PreferredPresenceUntil().Unix()
		prefPresenceUntil = &t
	}

	var verifiedAt *int64
	if u.VerifiedAt() != nil {
		t := u.VerifiedAt().Unix()
		verifiedAt = &t
	}

	var disabledAt *int64
	if u.DisabledAt() != nil {
		t := u.DisabledAt().Unix()
		disabledAt = &t
	}

	var deleteScheduledAt *int64
	if u.DeleteScheduledAt() != nil {
		t := u.DeleteScheduledAt().Unix()
		deleteScheduledAt = &t
	}

	return &UserAggregate{
		ID:                     u.ID().UUID(),
		Username:               u.Username().String(),
		DisplayName:            p.DisplayName().String(),
		Bio:                    bio,
		AvatarURL:              avatarURL,
		BannerColor:            bannerColor,
		PreferredPresence:      prefPresence,
		PreferredPresenceUntil: prefPresenceUntil,
		VerifiedAt:             verifiedAt,
		DisabledAt:             disabledAt,
		DeleteScheduledAt:      deleteScheduledAt,
		CreatedAt:              u.CreatedAt().Unix(),
		UpdatedAt:              u.UpdatedAt().Unix(),
	}
}

func (dto *UserAggregate) Reconstitute() (*user.Aggregate, error) {
	if dto == nil {
		return nil, nil
	}

	id, err := user.NewID(dto.ID)
	if err != nil {
		return nil, err
	}

	username, err := user.NewUsername(dto.Username)
	if err != nil {
		return nil, err
	}

	displayName, err := user.NewDisplayName(dto.DisplayName)
	if err != nil {
		return nil, err
	}

	var bio *user.Bio
	if dto.Bio != "" {
		b, err := user.NewBio(&dto.Bio)
		if err != nil {
			return nil, err
		}
		bio = &b
	}

	var avatarURL *user.URL
	if dto.AvatarURL != "" {
		u, err := user.NewURL(&dto.AvatarURL)
		if err != nil {
			return nil, err
		}
		avatarURL = &u
	}

	var bannerColor *user.BannerColor
	if dto.BannerColor != "" {
		bc, err := user.NewBannerColor(&dto.BannerColor)
		if err != nil {
			return nil, err
		}
		bannerColor = &bc
	}

	var prefPresence *user.PreferredPresence
	if dto.PreferredPresence != 0 {
		pp, err := user.NewPreferredPresenceFromInt16(dto.PreferredPresence)
		if err != nil {
			return nil, err
		}
		prefPresence = &pp
	}

	var prefPresenceUntil *time.Time
	if dto.PreferredPresenceUntil != nil {
		t := time.Unix(*dto.PreferredPresenceUntil, 0).UTC()
		prefPresenceUntil = &t
	}

	var verifiedAt *time.Time
	if dto.VerifiedAt != nil {
		t := time.Unix(*dto.VerifiedAt, 0).UTC()
		verifiedAt = &t
	}

	var disabledAt *time.Time
	if dto.DisabledAt != nil {
		t := time.Unix(*dto.DisabledAt, 0).UTC()
		disabledAt = &t
	}

	var deleteScheduledAt *time.Time
	if dto.DeleteScheduledAt != nil {
		t := time.Unix(*dto.DeleteScheduledAt, 0).UTC()
		deleteScheduledAt = &t
	}

	// Reconstitute domain entities
	u := user.Reconstitute(
		id,
		user.Email{}, // Empty email if omitted from public aggregate cache
		username,
		nil, // Phone omitted
		user.Password{},
		prefPresence,
		prefPresenceUntil,
		verifiedAt,
		disabledAt,
		deleteScheduledAt,
		time.Unix(dto.CreatedAt, 0).UTC(),
		time.Unix(dto.UpdatedAt, 0).UTC(),
	)

	p := user.ReconstituteProfile(
		id,
		displayName,
		bio,
		avatarURL,
		bannerColor,
		time.Unix(dto.UpdatedAt, 0).UTC(),
	)

	return user.NewAggregate(u, p), nil
}

func userAggregateKey(userID uuid.UUID) string {
	return fmt.Sprintf("user:%s:aggregate", userID.String())
}

func (u *UserCache) SetAggregate(ctx context.Context, agg *user.Aggregate) error {
	if agg == nil {
		return nil
	}
	return u.SetAggregateBatch(ctx, []*user.Aggregate{agg})
}

func (u *UserCache) SetAggregateBatch(ctx context.Context, aggs []*user.Aggregate) error {
	if len(aggs) == 0 {
		return nil
	}

	dtos := make([]*UserAggregate, 0, len(aggs))
	for _, agg := range aggs {
		if dto := NewUserAggregate(agg); dto != nil {
			dtos = append(dtos, dto)
		}
	}

	if len(dtos) == 0 {
		return nil
	}

	err := u.store.ExecPipelineFunc(ctx, func(pipe goredis.Pipeliner) error {
		for _, dto := range dtos {
			key := userAggregateKey(dto.ID)
			fields := buildUserAggregateFields(dto)

			pipe.HSet(ctx, key, fields)
			pipe.Expire(ctx, key, u.ttl)
		}
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}

	return nil
}

func (u *UserCache) GetAggregate(ctx context.Context, userID uuid.UUID) (*user.Aggregate, error) {
	found, _, err := u.GetAggregateBatch(ctx, []uuid.UUID{userID})
	if err != nil {
		return nil, err
	}

	return found[userID], nil
}

func (u *UserCache) GetAggregateBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*user.Aggregate, []uuid.UUID, error) {
	if len(userIDs) == 0 {
		return make(map[uuid.UUID]*user.Aggregate), nil, nil
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
			key := userAggregateKey(id)
			cmds[id] = pipe.HGetAll(ctx, key)
		}
		return nil
	})
	if err != nil {
		return nil, nil, redis.NewError(err, redis.ScopeUser)
	}

	found := make(map[uuid.UUID]*user.Aggregate, len(uniqueIDs))
	var missing []uuid.UUID

	for id, cmd := range cmds {
		res, resErr := cmd.Result()
		if resErr != nil || len(res) == 0 {
			missing = append(missing, id)
			continue
		}

		dto, parseErr := parseUserAggregateDTO(res)
		if parseErr != nil {
			missing = append(missing, id)
			continue
		}

		domainAgg, domainErr := dto.Reconstitute()
		if domainErr != nil || domainAgg == nil {
			missing = append(missing, id)
			continue
		}

		found[id] = domainAgg
	}

	return found, missing, nil
}

func buildUserAggregateFields(agg *UserAggregate) map[string]interface{} {
	fields := map[string]interface{}{
		"id":                 agg.ID.String(),
		"username":           agg.Username,
		"display_name":       agg.DisplayName,
		"bio":                agg.Bio,
		"avatar_url":         agg.AvatarURL,
		"banner_color":       agg.BannerColor,
		"preferred_presence": strconv.Itoa(int(agg.PreferredPresence)),
		"created_at":         strconv.FormatInt(agg.CreatedAt, 10),
		"updated_at":         strconv.FormatInt(agg.UpdatedAt, 10),
	}

	if agg.PreferredPresenceUntil != nil {
		fields["preferred_presence_until"] = strconv.FormatInt(*agg.PreferredPresenceUntil, 10)
	}
	if agg.VerifiedAt != nil {
		fields["verified_at"] = strconv.FormatInt(*agg.VerifiedAt, 10)
	}
	if agg.DisabledAt != nil {
		fields["disabled_at"] = strconv.FormatInt(*agg.DisabledAt, 10)
	}
	if agg.DeleteScheduledAt != nil {
		fields["delete_scheduled_at"] = strconv.FormatInt(*agg.DeleteScheduledAt, 10)
	}

	return fields
}

func parseUserAggregateDTO(m map[string]string) (*UserAggregate, error) {
	id, err := uuid.Parse(m["id"])
	if err != nil {
		return nil, err
	}

	prefPresenceInt, _ := strconv.ParseInt(m["preferred_presence"], 10, 16)
	createdAt, _ := strconv.ParseInt(m["created_at"], 10, 64)
	updatedAt, _ := strconv.ParseInt(m["updated_at"], 10, 64)

	agg := &UserAggregate{
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
			agg.PreferredPresenceUntil = &ts
		}
	}
	if val, ok := m["verified_at"]; ok && val != "" {
		if ts, pErr := strconv.ParseInt(val, 10, 64); pErr == nil {
			agg.VerifiedAt = &ts
		}
	}
	if val, ok := m["disabled_at"]; ok && val != "" {
		if ts, pErr := strconv.ParseInt(val, 10, 64); pErr == nil {
			agg.DisabledAt = &ts
		}
	}
	if val, ok := m["delete_scheduled_at"]; ok && val != "" {
		if ts, pErr := strconv.ParseInt(val, 10, 64); pErr == nil {
			agg.DeleteScheduledAt = &ts
		}
	}

	return agg, nil
}
