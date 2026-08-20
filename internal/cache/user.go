package cache

import (
	"context"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/user"

	redisdriver "github.com/redis/go-redis/v9"
)

type UserCache struct {
	*KeyCache[fields.ID, User]
	ttl time.Duration
}

func NewUserCache(client redisdriver.Cmdable, ttl time.Duration) *UserCache {
	return &UserCache{
		KeyCache: NewKeyCache[fields.ID, User](client, redis.ScopeUser, UserKey),
		ttl:      ttl,
	}
}

func (u *UserCache) Get(ctx context.Context, id fields.ID) (*user.User, error) {
	dto, err := u.KeyCache.Get(ctx, id)
	if err != nil || dto == nil {
		return nil, err
	}

	return dto.ToDomain()
}

func (u *UserCache) GetBatch(
	ctx context.Context,
	ids []fields.ID,
) (map[fields.ID]*user.User, []fields.ID, error) {
	dtos, missing, err := u.KeyCache.GetBatch(ctx, ids)
	if err != nil {
		return nil, nil, err
	}

	found := make(map[fields.ID]*user.User, len(dtos))
	for id, dto := range dtos {
		if dto == nil {
			missing = append(missing, id)
			continue
		}

		usr, err := dto.ToDomain()
		if err != nil {
			missing = append(missing, id)
			continue
		}
		found[id] = usr
	}

	return found, missing, nil
}

func (u *UserCache) Set(ctx context.Context, usr *user.User) error {
	if usr == nil {
		return nil
	}
	return u.KeyCache.Set(ctx, usr.ID(), ParseUser(usr), u.ttl)
}

func (u *UserCache) SetBatch(ctx context.Context, users []*user.User) error {
	if len(users) == 0 {
		return nil
	}

	dtos := make(map[fields.ID]User, len(users))
	for _, usr := range users {
		if usr == nil {
			continue
		}
		dtos[usr.ID()] = ParseUser(usr)
	}

	return u.KeyCache.SetBatch(ctx, dtos, u.ttl)
}

func (u *UserCache) Delete(ctx context.Context, id fields.ID) error {
	if id.IsZero() {
		return nil
	}
	return u.KeyCache.Delete(ctx, id)
}

func (u *UserCache) DeleteBatch(ctx context.Context, ids []fields.ID) error {
	if len(ids) == 0 {
		return nil
	}
	return u.KeyCache.DeleteBatch(ctx, ids)
}

func (u *UserCache) AddChannelIDs(ctx context.Context, userID fields.ID, channelIDs ...fields.ID) error {
	if userID.IsZero() || len(channelIDs) == 0 {
		return nil
	}

	members := make([]any, 0, len(channelIDs))
	for _, id := range channelIDs {
		if !id.IsZero() {
			members = append(members, id.String())
		}
	}

	if len(members) == 0 {
		return nil
	}

	key := UserChannelIDsKey(userID)

	// SADD and refresh TTL
	pipe := u.KeyCache.client.Pipeline()
	pipe.SAdd(ctx, key, members...)
	pipe.Expire(ctx, key, u.ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}

	return nil
}

func (u *UserCache) RemoveChannelID(ctx context.Context, userID fields.ID, channelID fields.ID) error {
	if userID.IsZero() || channelID.IsZero() {
		return nil
	}

	key := UserChannelIDsKey(userID)
	if err := u.KeyCache.client.SRem(ctx, key, channelID.String()).Err(); err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}

	return nil
}

func (u *UserCache) RemoveChannelIDBatch(
	ctx context.Context,
	userIDs []fields.ID,
	channelID fields.ID,
) error {
	if len(userIDs) == 0 || channelID.IsZero() {
		return nil
	}

	channelIDStr := channelID.String()
	pipe := u.KeyCache.client.Pipeline()

	var queuedCount int
	for _, id := range userIDs {
		if !id.IsZero() {
			key := UserChannelIDsKey(id)
			pipe.SRem(ctx, key, channelIDStr)
			queuedCount++
		}
	}

	if queuedCount == 0 {
		return nil
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}

	return nil
}

func (u *UserCache) GetChannelIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error) {
	if userID.IsZero() {
		return nil, nil
	}

	key := UserChannelIDsKey(userID)
	rawIDs, err := u.KeyCache.client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, redis.NewError(err, redis.ScopeUser)
	}

	if len(rawIDs) == 0 {
		return nil, nil
	}

	parsedIDs := make([]fields.ID, 0, len(rawIDs))
	for _, raw := range rawIDs {
		id, err := fields.ParseIDFromString("channel_id", raw)
		if err != nil {
			continue
		}
		parsedIDs = append(parsedIDs, id)
	}

	return parsedIDs, nil
}
