package cache

import (
	"context"
	"encoding/json"
	"time"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/user"

	redisdriver "github.com/redis/go-redis/v9"
)

type UserCache struct {
	client redisdriver.Cmdable
	scope  redis.Scope
	ttl    time.Duration
}

func NewUserCache(
	client redisdriver.Cmdable,
	scope redis.Scope,
	ttl time.Duration,
) *UserCache {
	return &UserCache{
		client: client,
		scope:  scope,
		ttl:    ttl,
	}
}

func (u *UserCache) Get(ctx context.Context, id fields.ID) (*user.User, error) {
	redisKey := UserKey(id)

	data, err := u.client.Get(ctx, redisKey).Bytes()
	if redis.IsCacheMiss(err) {
		return nil, nil
	}
	if err != nil {
		return nil, redis.NewError(err, u.scope)
	}

	var dto User
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, errs.Internal("Failed to unmarshal cached user.").
			Meta("scope", u.scope.String()).
			Wrap(err)
	}

	return dto.ToDomain()
}

func (u *UserCache) GetBatch(
	ctx context.Context,
	ids []fields.ID,
) (map[fields.ID]*user.User, []fields.ID, error) {
	found := make(map[fields.ID]*user.User, len(ids))
	var missing []fields.ID

	for i := 0; i < len(ids); i += KeyMaxBatchSize {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		end := min(i+KeyMaxBatchSize, len(ids))
		chunk := ids[i:end]

		redisKeys := make([]string, len(chunk))
		for j, id := range chunk {
			redisKeys[j] = UserKey(id)
		}

		vals, err := u.client.MGet(ctx, redisKeys...).Result()
		if err != nil {
			return nil, nil, redis.NewError(err, u.scope)
		}

		for j, raw := range vals {
			id := chunk[j]

			if raw == nil {
				missing = append(missing, id)
				continue
			}

			var data []byte
			switch v := raw.(type) {
			case string:
				if v == "" {
					missing = append(missing, id)
					continue
				}
				data = []byte(v)
			case []byte:
				if len(v) == 0 {
					missing = append(missing, id)
					continue
				}
				data = v
			default:
				missing = append(missing, id)
				continue
			}

			var dto User
			if err := json.Unmarshal(data, &dto); err != nil {
				return nil, nil, errs.Internal("Failed to unmarshal cached user batch item.").
					Meta("scope", u.scope.String()).
					Wrap(err)
			}

			usr, err := dto.ToDomain()
			if err != nil {
				missing = append(missing, id)
				continue
			}

			found[id] = usr
		}
	}

	if len(missing) > 0 {
		seen := make(map[fields.ID]struct{}, len(missing))
		result := make([]fields.ID, 0, len(missing))

		for _, id := range missing {
			if _, inFound := found[id]; inFound {
				continue
			}
			if _, inSeen := seen[id]; !inSeen {
				seen[id] = struct{}{}
				result = append(result, id)
			}
		}
		missing = result
	}

	return found, missing, nil
}

func (u *UserCache) Set(ctx context.Context, usr *user.User) error {
	redisKey := UserKey(usr.ID())
	dto := ParseUser(usr)

	bytes, err := json.Marshal(dto)
	if err != nil {
		return errs.Internal("Failed to marshal user json.").
			Meta("scope", u.scope.String()).
			Wrap(err)
	}

	if err := u.client.Set(ctx, redisKey, bytes, u.ttl).Err(); err != nil {
		return redis.NewError(err, u.scope)
	}

	return nil
}

func (u *UserCache) SetBatch(ctx context.Context, users []*user.User) error {
	validUsers := make([]*user.User, 0, len(users))
	for _, usr := range users {
		if usr != nil && !usr.ID().IsZero() {
			validUsers = append(validUsers, usr)
		}
	}

	if len(validUsers) == 0 {
		return nil
	}

	for i := 0; i < len(validUsers); i += KeyMaxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+KeyMaxBatchSize, len(validUsers))
		chunk := validUsers[i:end]

		pipe := u.client.Pipeline()
		for _, usr := range chunk {
			bytes, err := json.Marshal(ParseUser(usr))
			if err != nil {
				return errs.Internal("Failed to marshal user json.").
					Meta("scope", u.scope.String()).
					Wrap(err)
			}
			pipe.Set(ctx, UserKey(usr.ID()), bytes, u.ttl)
		}

		if _, err := pipe.Exec(ctx); err != nil {
			return redis.NewError(err, u.scope)
		}
	}

	return nil
}

func (u *UserCache) Delete(ctx context.Context, id fields.ID) error {
	if err := u.client.Del(ctx, UserKey(id)).Err(); err != nil {
		return redis.NewError(err, u.scope)
	}

	return nil
}

func (u *UserCache) DeleteBatch(ctx context.Context, ids []fields.ID) error {
	for i := 0; i < len(ids); i += KeyMaxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+KeyMaxBatchSize, len(ids))
		chunk := ids[i:end]

		redisKeys := make([]string, len(chunk))
		for j, id := range chunk {
			redisKeys[j] = UserKey(id)
		}

		if err := u.client.Del(ctx, redisKeys...).Err(); err != nil {
			return redis.NewError(err, u.scope)
		}
	}

	return nil
}

func (u *UserCache) GetChannelIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error) {
	key := UserChannelIDsKey(userID)
	rawIDs, err := u.client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, redis.NewError(err, u.scope)
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

func (u *UserCache) AddChannelIDs(ctx context.Context, userID fields.ID, channelIDs []fields.ID) error {
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

	pipe := u.client.Pipeline()
	pipe.SAdd(ctx, key, members...)
	pipe.Expire(ctx, key, u.ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, u.scope)
	}

	return nil
}

func (u *UserCache) RemoveChannelIDs(ctx context.Context, userID fields.ID, channelIDs []fields.ID) error {
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

	if err := u.client.SRem(ctx, key, members...).Err(); err != nil {
		return redis.NewError(err, u.scope)
	}

	return nil
}

func (u *UserCache) AddBatchChannelID(
	ctx context.Context,
	userIDs []fields.ID,
	channelID fields.ID,
) error {
	if len(userIDs) == 0 || channelID.IsZero() {
		return nil
	}

	channelIDStr := channelID.String()
	pipe := u.client.Pipeline()

	var queuedCount int
	for _, id := range userIDs {
		if !id.IsZero() {
			key := UserChannelIDsKey(id)
			pipe.SAdd(ctx, key, channelIDStr)
			pipe.Expire(ctx, key, u.ttl)
			queuedCount++
		}
	}

	if queuedCount == 0 {
		return nil
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, u.scope)
	}

	return nil
}

func (u *UserCache) RemoveBatchChannelID(
	ctx context.Context,
	userIDs []fields.ID,
	channelID fields.ID,
) error {
	if len(userIDs) == 0 || channelID.IsZero() {
		return nil
	}

	channelIDStr := channelID.String()
	pipe := u.client.Pipeline()

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
		return redis.NewError(err, u.scope)
	}

	return nil
}
