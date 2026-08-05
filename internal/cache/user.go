package cache

import (
	"bonfire-api/internal/redis"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type User struct {
	store redis.Store
	ttl   time.Duration
}

func NewUser(store redis.Store, ttl time.Duration) *User {
	return &User{
		store: store,
		ttl:   ttl,
	}
}

func userAggregateKey(userID uuid.UUID) string {
	return fmt.Sprintf("user:%s:aggregate", userID.String())
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
	CreatedAt              int64     `redis:"created_at"` // Unix ms
	UpdatedAt              int64     `redis:"updated_at"` // Unix ms
}

// SetUserAggregate populates or refreshes the user aggregate Hash with sliding TTL.
func (u *User) SetUserAggregate(ctx context.Context, agg *UserAggregate) error {
	if agg == nil {
		return nil
	}

	key := userAggregateKey(agg.ID)

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

	err := u.store.ExecPipelineFunc(ctx, func(pipe goredis.Pipeliner) error {
		pipe.HSet(ctx, key, fields)
		pipe.Expire(ctx, key, u.ttl)
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}

	return nil
}

// GetUserAggregate fetches the user aggregate and refreshes its sliding TTL on read.
func (u *User) GetUserAggregate(ctx context.Context, userID uuid.UUID) (*UserAggregate, error) {
	key := userAggregateKey(userID)

	var hGetAllCmd *goredis.MapStringStringCmd
	err := u.store.ExecPipelineFunc(ctx, func(pipe goredis.Pipeliner) error {
		hGetAllCmd = pipe.HGetAll(ctx, key)
		pipe.Expire(ctx, key, u.ttl)
		return nil
	})
	if err != nil {
		return nil, redis.NewError(err, redis.ScopeUser)
	}

	res, err := hGetAllCmd.Result()
	if err != nil || len(res) == 0 {
		return nil, nil // Cache Miss
	}

	return parseUserAggregateDTO(res)
}

// Helper to construct UserAggregate from HGETALL map output
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
