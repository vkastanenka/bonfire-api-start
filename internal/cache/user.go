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

// SetUserAggregate populates or refreshes a single user aggregate Hash with sliding TTL.
func (u *User) SetUserAggregate(ctx context.Context, agg *UserAggregate) error {
	if agg == nil {
		return nil
	}
	return u.SetUserAggregatesBatch(ctx, []*UserAggregate{agg})
}

// SetUserAggregatesBatch pipelines multiple user aggregates into Redis with sliding TTL.
func (u *User) SetUserAggregatesBatch(ctx context.Context, aggs []*UserAggregate) error {
	if len(aggs) == 0 {
		return nil
	}

	err := u.store.ExecPipelineFunc(ctx, func(pipe goredis.Pipeliner) error {
		for _, agg := range aggs {
			if agg == nil {
				continue
			}
			key := userAggregateKey(agg.ID)
			fields := buildUserAggregateFields(agg)

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

// GetUserAggregate fetches the user aggregate and refreshes its sliding TTL on read.
func (u *User) GetUserAggregate(ctx context.Context, userID uuid.UUID) (*UserAggregate, error) {
	found, _, err := u.GetUserAggregatesBatch(ctx, []uuid.UUID{userID})
	if err != nil {
		return nil, err
	}

	return found[userID], nil
}

// GetUserAggregatesBatch retrieves multiple user aggregates in a single Redis pipeline call.
// Returns a map of found users, and a slice of missing user UUIDs for DB backfill.
func (u *User) GetUserAggregatesBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*UserAggregate, []uuid.UUID, error) {
	if len(userIDs) == 0 {
		return make(map[uuid.UUID]*UserAggregate), nil, nil
	}

	// Deduplicate incoming IDs to prevent redundant Redis commands
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
			pipe.Expire(ctx, key, u.ttl)
		}
		return nil
	})
	if err != nil {
		return nil, nil, redis.NewError(err, redis.ScopeUser)
	}

	found := make(map[uuid.UUID]*UserAggregate, len(uniqueIDs))
	var missing []uuid.UUID

	for id, cmd := range cmds {
		res, resErr := cmd.Result()
		if resErr != nil || len(res) == 0 {
			missing = append(missing, id)
			continue
		}

		agg, parseErr := parseUserAggregateDTO(res)
		if parseErr != nil {
			missing = append(missing, id)
			continue
		}

		found[id] = agg
	}

	return found, missing, nil
}

// Helper to convert UserAggregate struct into a Redis HSET map
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
