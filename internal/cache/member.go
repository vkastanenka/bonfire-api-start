package cache

import (
	"bonfire-api/internal/redis"
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

var (
	ErrBatchSizeExceeded = errors.New("channel member batch size exceeds maximum limit of 10")
)

const MaxGroupMembers = 10

type ChannelMember struct {
	store redis.Store
	ttl   time.Duration
}

type ChannelMemberDTO struct {
	ChannelID         uuid.UUID  `redis:"channel_id"`
	UserID            uuid.UUID  `redis:"user_id"`
	LastReadAt        int64      `redis:"last_read_at"` // Unix ms
	LastReadMessageID *uuid.UUID `redis:"last_read_message_id"`
	PinnedAt          *int64     `redis:"pinned_at"`   // Unix ms
	MutedUntil        *int64     `redis:"muted_until"` // Unix ms
	IsVisible         bool       `redis:"is_visible"`
	CreatedAt         int64      `redis:"created_at"` // Unix ms
	UpdatedAt         int64      `redis:"updated_at"` // Unix ms
}

func NewChannelMember(store redis.Store, ttl time.Duration) *ChannelMember {
	return &ChannelMember{store: store, ttl: ttl}
}

func channelMemberKey(channelID, userID uuid.UUID) string {
	return fmt.Sprintf("channel:{%s}:member:%s", channelID.String(), userID.String())
}

// SetBatch populates or refreshes multiple per-user channel member state Hashes,
// strictly enforced up to the maximum Group DM size of 10 members.
func (c *ChannelMember) SetBatch(ctx context.Context, members []*ChannelMemberDTO) error {
	if len(members) == 0 {
		return nil
	}

	if len(members) > MaxGroupMembers {
		return fmt.Errorf("%w: received %d members", ErrBatchSizeExceeded, len(members))
	}

	err := c.store.ExecPipelineFunc(ctx, func(pipe goredis.Pipeliner) error {
		for _, member := range members {
			if member == nil {
				continue
			}

			key := channelMemberKey(member.ChannelID, member.UserID)

			fields := map[string]interface{}{
				"channel_id":   member.ChannelID.String(),
				"user_id":      member.UserID.String(),
				"last_read_at": strconv.FormatInt(member.LastReadAt, 10),
				"is_visible":   strconv.FormatBool(member.IsVisible),
				"created_at":   strconv.FormatInt(member.CreatedAt, 10),
				"updated_at":   strconv.FormatInt(member.UpdatedAt, 10),
			}

			if member.LastReadMessageID != nil {
				fields["last_read_message_id"] = member.LastReadMessageID.String()
			}
			if member.PinnedAt != nil {
				fields["pinned_at"] = strconv.FormatInt(*member.PinnedAt, 10)
			}
			if member.MutedUntil != nil {
				fields["muted_until"] = strconv.FormatInt(*member.MutedUntil, 10)
			}

			pipe.HSet(ctx, key, fields)
			pipe.Expire(ctx, key, c.ttl)
		}
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopeChannel)
	}

	return nil
}

// Get fetches shared channel metadata and refreshes its sliding TTL on read.
func (c *ChannelMember) Get(ctx context.Context, channelID, userID uuid.UUID) (*ChannelMemberDTO, error) {
	key := channelMemberKey(channelID, userID)

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

	return parseChannelMemberDTO(res)
}

// TouchLastRead updates last_read_at, last_read_message_id, and updated_at when a user reads messages in a channel.
func (c *ChannelMember) TouchLastRead(ctx context.Context, channelID, userID, messageID uuid.UUID, readAt int64) error {
	key := channelMemberKey(channelID, userID)

	err := c.store.ExecPipelineFunc(ctx, func(pipe goredis.Pipeliner) error {
		pipe.HSet(ctx, key, map[string]interface{}{
			"last_read_message_id": messageID.String(),
			"last_read_at":         strconv.FormatInt(readAt, 10),
			"updated_at":           strconv.FormatInt(readAt, 10),
		})
		pipe.Expire(ctx, key, c.ttl)
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopeChannel)
	}

	return nil
}

// Helper to construct ChannelMemberDTO from HGETALL map output
func parseChannelMemberDTO(m map[string]string) (*ChannelMemberDTO, error) {
	channelID, err := uuid.Parse(m["channel_id"])
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(m["user_id"])
	if err != nil {
		return nil, err
	}

	lastReadAt, _ := strconv.ParseInt(m["last_read_at"], 10, 64)
	isVisible, _ := strconv.ParseBool(m["is_visible"])
	createdAt, _ := strconv.ParseInt(m["created_at"], 10, 64)
	updatedAt, _ := strconv.ParseInt(m["updated_at"], 10, 64)

	dto := &ChannelMemberDTO{
		ChannelID:  channelID,
		UserID:     userID,
		LastReadAt: lastReadAt,
		IsVisible:  isVisible,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}

	if val, ok := m["last_read_message_id"]; ok && val != "" {
		if msgID, pErr := uuid.Parse(val); pErr == nil {
			dto.LastReadMessageID = &msgID
		}
	}
	if val, ok := m["pinned_at"]; ok && val != "" {
		if ts, pErr := strconv.ParseInt(val, 10, 64); pErr == nil {
			dto.PinnedAt = &ts
		}
	}
	if val, ok := m["muted_until"]; ok && val != "" {
		if ts, pErr := strconv.ParseInt(val, 10, 64); pErr == nil {
			dto.MutedUntil = &ts
		}
	}

	return dto, nil
}
