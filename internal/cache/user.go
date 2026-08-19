package cache

import (
	"context"
	"encoding/json"
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

func (u *UserCache) SetChannelIDs(ctx context.Context, userID fields.ID, channelIDs []fields.ID) error {
	if userID.IsZero() || len(channelIDs) == 0 {
		return nil
	}

	key := UserChannelIDsKey(userID)

	rawIDs := make([]string, 0, len(channelIDs))
	for _, id := range channelIDs {
		if !id.IsZero() {
			rawIDs = append(rawIDs, id.String())
		}
	}

	if len(rawIDs) == 0 {
		return nil
	}

	data, err := json.Marshal(rawIDs)
	if err != nil {
		return err
	}

	if err := u.KeyCache.client.Set(ctx, key, data, u.ttl).Err(); err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}

	return nil
}

func (u *UserCache) DeleteChannelIDs(ctx context.Context, userID fields.ID) error {
	if userID.IsZero() {
		return nil
	}

	key := UserChannelIDsKey(userID)
	if err := u.KeyCache.client.Del(ctx, key).Err(); err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}

	return nil
}

func (u *UserCache) DeleteChannelIDsBatch(ctx context.Context, userIDs []fields.ID) error {
	if len(userIDs) == 0 {
		return nil
	}

	keys := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		if !id.IsZero() {
			keys = append(keys, UserChannelIDsKey(id))
		}
	}

	if len(keys) == 0 {
		return nil
	}

	if err := u.KeyCache.client.Del(ctx, keys...).Err(); err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}

	return nil
}
