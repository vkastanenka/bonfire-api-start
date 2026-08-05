package cache

import (
	"bonfire-api/internal/redis"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type Channel struct {
	store redis.Store
	ttl   time.Duration
}

type ChannelDTO struct {
	ID            uuid.UUID  `redis:"id"`
	Type          int16      `redis:"type"`
	Name          *string    `redis:"name"`
	IconURL       *string    `redis:"icon_url"`
	MemberCount   int        `redis:"member_count"`
	MemberIDs     string     `redis:"member_ids"`
	LastMessageID *uuid.UUID `redis:"last_message_id"`
	LastMessageAt *int64     `redis:"last_message_at"`
	CreatedAt     int64      `redis:"created_at"` // Unix ms
	UpdatedAt     int64      `redis:"updated_at"` // Unix ms
}

func NewChannel(store redis.Store, ttl time.Duration) *Channel {
	return &Channel{store: store, ttl: ttl}
}

func channelKey(id uuid.UUID) string {
	return fmt.Sprintf("channel:{%s}", id.String())
}

// Set populates or refreshes the shared channel metadata Hash with sliding TTL.
func (c *Channel) Set(ctx context.Context, ch *ChannelDTO) error {
	key := channelKey(ch.ID)

	fields := map[string]interface{}{
		"id":           ch.ID.String(),
		"type":         strconv.Itoa(int(ch.Type)),
		"member_count": strconv.Itoa(ch.MemberCount),
		"member_ids":   ch.MemberIDs,
		"created_at":   strconv.FormatInt(ch.CreatedAt, 10),
		"updated_at":   strconv.FormatInt(ch.UpdatedAt, 10),
	}

	if ch.Name != nil {
		fields["name"] = *ch.Name
	}
	if ch.IconURL != nil {
		fields["icon_url"] = *ch.IconURL
	}
	if ch.LastMessageID != nil {
		fields["last_message_id"] = ch.LastMessageID.String()
	}
	if ch.LastMessageAt != nil {
		fields["last_message_at"] = strconv.FormatInt(*ch.LastMessageAt, 10)
	}

	err := c.store.ExecPipelineFunc(ctx, func(pipe goredis.Pipeliner) error {
		pipe.HSet(ctx, key, fields)
		pipe.Expire(ctx, key, c.ttl)
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopeChannel)
	}

	return nil
}

// Get fetches shared channel metadata and refreshes its sliding TTL on read.
func (c *Channel) Get(ctx context.Context, channelID uuid.UUID) (*ChannelDTO, error) {
	key := channelKey(channelID)

	var hGetAllCmd *goredis.MapStringStringCmd
	err := c.store.ExecPipelineFunc(ctx, func(pipe goredis.Pipeliner) error {
		hGetAllCmd = pipe.HGetAll(ctx, key)
		pipe.Expire(ctx, key, c.ttl)
		return nil
	})
	if err != nil {
		return nil, redis.NewError(err, redis.ScopeChannel)
	}

	res, err := hGetAllCmd.Result()
	if err != nil || len(res) == 0 {
		return nil, nil // Cache Miss
	}

	return parseChannelDTO(res)
}

// TouchLastMessage updates last_message_id, last_message_at, and updated_at atomically on message dispatch.
func (c *Channel) TouchLastMessage(ctx context.Context, channelID uuid.UUID, messageID uuid.UUID, timestamp int64) error {
	key := channelKey(channelID)

	err := c.store.ExecPipelineFunc(ctx, func(pipe goredis.Pipeliner) error {
		pipe.HSet(ctx, key, map[string]interface{}{
			"last_message_id": messageID.String(),
			"last_message_at": strconv.FormatInt(timestamp, 10),
			"updated_at":      strconv.FormatInt(timestamp, 10),
		})
		pipe.Expire(ctx, key, c.ttl)
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopeChannel)
	}

	return nil
}

// Helper to construct DTO from HGETALL map output
func parseChannelDTO(m map[string]string) (*ChannelDTO, error) {
	id, err := uuid.Parse(m["id"])
	if err != nil {
		return nil, err
	}

	typeInt, _ := strconv.ParseInt(m["type"], 10, 16)
	memberCount, _ := strconv.Atoi(m["member_count"])
	createdAt, _ := strconv.ParseInt(m["created_at"], 10, 64)
	updatedAt, _ := strconv.ParseInt(m["updated_at"], 10, 64)

	dto := &ChannelDTO{
		ID:          id,
		Type:        int16(typeInt),
		MemberCount: memberCount,
		MemberIDs:   m["member_ids"],
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	if val, ok := m["name"]; ok && val != "" {
		dto.Name = &val
	}
	if val, ok := m["icon_url"]; ok && val != "" {
		dto.IconURL = &val
	}
	if val, ok := m["last_message_id"]; ok && val != "" {
		if msgID, pErr := uuid.Parse(val); pErr == nil {
			dto.LastMessageID = &msgID
		}
	}
	if val, ok := m["last_message_at"]; ok && val != "" {
		if ts, pErr := strconv.ParseInt(val, 10, 64); pErr == nil {
			dto.LastMessageAt = &ts
		}
	}

	return dto, nil
}

// FormatMemberIDs converts a slice of member UUIDs into a CSV string for `member_ids`.
func FormatMemberIDs(memberIDs []uuid.UUID) string {
	strs := make([]string, len(memberIDs))
	for i, id := range memberIDs {
		strs[i] = id.String()
	}
	return strings.Join(strs, ",")
}
