package user

import (
	"bonfire-api/internal/presence"
	"bonfire-api/internal/redis"
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

const (
	CacheTTL = 24 * time.Hour
)

func CacheKey(id uuid.UUID) string { return "user:" + "{" + id.String() + "}" }

type UserCache struct {
	client redisdriver.Cmdable
}

func NewUserCache(client redisdriver.Cmdable) *UserCache {
	return &UserCache{client: client}
}

func (c *UserCache) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	return redis.GetAndUnmarshal(
		ctx,
		c.client,
		redis.ResourceUser,
		CacheKey(id),
		UnmarshalCacheDTO,
	)
}

func (c *UserCache) Set(ctx context.Context, u *User) error {
	if u == nil {
		return nil
	}

	return redis.MarshalAndSet(
		ctx,
		c.client,
		redis.ResourceUser,
		CacheKey(u.ID),
		u,
		CacheTTL,
		MarshalCacheDTO,
	)
}

func (c *UserCache) Delete(ctx context.Context, id uuid.UUID) error {
	if err := c.client.Del(ctx, CacheKey(id)).Err(); err != nil {
		return redis.NewError(err, redis.ResourceUser)
	}
	return nil
}

func (c *UserCache) GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*User, []uuid.UUID, error) {
	return redis.GetAndUnmarshalBatch(
		ctx,
		c.client,
		redis.ResourceUser,
		CacheKey,
		ids,
		UnmarshalCacheDTO,
	)
}

func (c *UserCache) SetBatch(ctx context.Context, users map[uuid.UUID]*User) error {
	return redis.MarshalAndSetBatch(
		ctx,
		c.client,
		redis.ResourceUser,
		CacheKey,
		users,
		CacheTTL,
		MarshalCacheDTO,
	)
}

func (c *UserCache) DeleteBatch(ctx context.Context, ids []uuid.UUID) error {
	return redis.DeleteBatch(ctx, c.client, redis.ResourceUser, CacheKey, ids)
}

type CacheDTO struct {
	ID                     uuid.UUID          `json:"id"`
	Email                  string             `json:"email"`
	Username               string             `json:"username"`
	DisplayName            string             `json:"display_name"`
	Phone                  *string            `json:"phone,omitempty"`
	Bio                    *string            `json:"bio,omitempty"`
	AvatarURL              *string            `json:"avatar_url,omitempty"`
	BannerColor            *string            `json:"banner_color,omitempty"`
	PreferredPresence      *presence.Presence `json:"preferred_presence,omitempty"`
	PreferredPresenceUntil *time.Time         `json:"preferred_presence_until,omitempty"`
	VerifiedAt             *time.Time         `json:"verified_at,omitempty"`
	DisabledAt             *time.Time         `json:"disabled_at,omitempty"`
	DeleteScheduledAt      *time.Time         `json:"delete_scheduled_at,omitempty"`
	CreatedAt              time.Time          `json:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at"`
}

func NewCacheDTO(u *User) CacheDTO {
	return CacheDTO{
		ID:                     u.ID,
		Email:                  u.Email,
		Username:               u.Username,
		DisplayName:            u.DisplayName,
		Phone:                  u.Phone,
		Bio:                    u.Bio,
		AvatarURL:              u.AvatarURL,
		BannerColor:            u.BannerColor,
		PreferredPresence:      u.PreferredPresence,
		PreferredPresenceUntil: u.PreferredPresenceUntil,
		VerifiedAt:             u.VerifiedAt,
		DisabledAt:             u.DisabledAt,
		DeleteScheduledAt:      u.DeleteScheduledAt,
		CreatedAt:              u.CreatedAt,
		UpdatedAt:              u.UpdatedAt,
	}
}

func (dto CacheDTO) ToDomain() *User {
	return Reconstitute(
		dto.ID,
		dto.Email,
		dto.Username,
		"",
		dto.Phone,
		dto.DisplayName,
		dto.Bio,
		dto.AvatarURL,
		dto.BannerColor,
		dto.PreferredPresence,
		dto.PreferredPresenceUntil,
		dto.VerifiedAt,
		dto.DisabledAt,
		dto.DeleteScheduledAt,
		dto.CreatedAt,
		dto.UpdatedAt,
	)
}

func UnmarshalCacheDTO(data []byte) (*User, error) {
	var dto CacheDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}
	return dto.ToDomain(), nil
}

func MarshalCacheDTO(u *User) ([]byte, error) {
	dto := NewCacheDTO(u)
	return json.Marshal(dto)
}
