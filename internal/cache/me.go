package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

// ErrMeCacheMiss indicates that the user's channel index key does not exist in Redis,
// signalling to the caller layer that a full PostgreSQL hydration/backfill is required.
var ErrMeCacheMiss = errors.New("me channels cache miss")

type Me struct {
	store redis.Store
	ttl   time.Duration
}

func NewMe(store redis.Store, ttl time.Duration) *Me {
	return &Me{
		store: store,
		ttl:   ttl,
	}
}

type MeChannel struct {
	ID      uuid.UUID       `json:"id"`
	Type    string          `json:"type"`
	Name    string          `json:"name"`
	IconURL string          `json:"iconUrl"`
	Peers   []MeChannelPeer `json:"peers,omitempty"`
}

type MeChannelIDs struct {
	ChannelIDs []string `json:"channelIds"`
	NextCursor int64    `json:"nextCursor,omitempty"`
	HasMore    bool     `json:"hasMore"`
}

type MeChannelIndexItem struct {
	ChannelID uuid.UUID
	Score     int64
}

type MeChannelPeer struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"displayName"`
	AvatarURL   string    `json:"avatarUrl"`
	Presence    string    `json:"presence"`
}

type MeChannelResult struct {
	Channels   []MeChannel `json:"channels"`
	NextCursor int64       `json:"nextCursor,omitempty"`
	HasMore    bool        `json:"hasMore"`
}

func meChannelsKey(userID uuid.UUID) string {
	return fmt.Sprintf("me:%s:channels", userID.String())
}

// SetMeChannels performs a full hydration of a user's channel ZSET index with a sliding TTL,
// strictly capped at the top 500 most recent/pinned channels.
func (m *Me) SetMeChannels(ctx context.Context, userID uuid.UUID, items []MeChannelIndexItem) error {
	key := meChannelsKey(userID)

	// Even if items is empty, we touch the key/TTL so Redis knows this user has 0 channels (cache hit)
	if len(items) == 0 {
		err := m.store.ExecPipelineFunc(ctx, func(pipe goredis.Pipeliner) error {
			// Touch an empty ZSET via Del or leave as empty with an explicit TTL marker if needed.
			// Pipeline a simple expire to ensure existence check works cleanly.
			pipe.Expire(ctx, key, m.ttl)
			return nil
		})
		if err != nil {
			return redis.NewError(err, redis.ScopeMe)
		}
		return nil
	}

	const maxChannels = 500
	if len(items) > maxChannels {
		items = items[:maxChannels]
	}

	zMembers := make([]goredis.Z, len(items))
	for i, item := range items {
		zMembers[i] = goredis.Z{
			Score:  float64(item.Score),
			Member: item.ChannelID.String(),
		}
	}

	err := m.store.ExecPipelineFunc(ctx, func(pipe goredis.Pipeliner) error {
		pipe.ZAdd(ctx, key, zMembers...)
		// Trim Redis ZSET to top 500
		pipe.ZRemRangeByRank(ctx, key, 0, -maxChannels-1)
		pipe.Expire(ctx, key, m.ttl)
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopeMe)
	}

	return nil
}

// GetMeChannelIDsBatch retrieves a reverse-chronological page of channel UUID strings.
// Differentiates between an empty channel list (valid hit) and a missing cache key (ErrMeCacheMiss).
func (m *Me) GetMeChannelIDsBatch(ctx context.Context, userID uuid.UUID, cursor int64, limit int64) (*MeChannelIDs, error) {
	key := meChannelsKey(userID)

	maxScore := "+inf"
	if cursor > 0 {
		maxScore = fmt.Sprintf("(%d", cursor)
	}

	var existsCmd *goredis.IntCmd
	var zCmd *goredis.ZSliceCmd

	err := m.store.ExecPipelineFunc(ctx, func(pipe goredis.Pipeliner) error {
		existsCmd = pipe.Exists(ctx, key)
		zCmd = pipe.ZRevRangeByScoreWithScores(ctx, key, &goredis.ZRangeBy{
			Min:    "-inf",
			Max:    maxScore,
			Offset: 0,
			Count:  limit + 1,
		})
		return nil
	})
	if err != nil {
		return nil, redis.NewError(err, redis.ScopeMe)
	}

	if existsCmd.Val() == 0 {
		return nil, ErrMeCacheMiss
	}

	zs, err := zCmd.Result()
	if err != nil {
		return nil, redis.NewError(err, redis.ScopeMe)
	}

	paged := &MeChannelIDs{
		ChannelIDs: make([]string, 0, limit),
		HasMore:    false,
	}

	if int64(len(zs)) > limit {
		paged.HasMore = true
		zs = zs[:limit]
	}

	for _, z := range zs {
		if channelID, ok := z.Member.(string); ok {
			paged.ChannelIDs = append(paged.ChannelIDs, channelID)
		}
	}

	if len(zs) > 0 {
		paged.NextCursor = int64(zs[len(zs)-1].Score)
	}

	return paged, nil
}

// GetMeChannelsBatch retrieves paginated channels and hydrates metadata, peer profiles, and presence.
// Returns ErrMeCacheMiss if the index ZSET is not cached or if channel metadata has expired.
func (m *Me) GetMeChannelsBatch(ctx context.Context, userID uuid.UUID, cursor int64, limit int64) (*MeChannelResult, error) {
	if limit <= 0 || limit > 50 {
		limit = 50
	}

	// 1. Fetch current page of channel UUIDs from ZSET
	pagedIDs, err := m.GetMeChannelIDsBatch(ctx, userID, cursor, limit)
	if err != nil {
		return nil, err // Returns ErrMeCacheMiss or wrapped redis error
	}

	if len(pagedIDs.ChannelIDs) == 0 {
		return &MeChannelResult{
			Channels: []MeChannel{},
			HasMore:  false,
		}, nil
	}

	// 2. Pipeline HGETALL for metadata across all channel IDs on this page
	cmdMap := make(map[string]*goredis.MapStringStringCmd, len(pagedIDs.ChannelIDs))
	err = m.store.ExecPipelineFunc(ctx, func(pipe goredis.Pipeliner) error {
		for _, cid := range pagedIDs.ChannelIDs {
			cUUID, pErr := uuid.Parse(cid)
			if pErr != nil {
				continue
			}
			cmdMap[cid] = pipe.HGetAll(ctx, channelKey(cUUID))
			// Refresh TTL on active channel metadata read
			pipe.Expire(ctx, channelKey(cUUID), m.ttl)
		}
		return nil
	})
	if err != nil {
		return nil, redis.NewError(err, redis.ScopeMe)
	}

	type parsedChannel struct {
		channel MeChannel
		peerIDs []string
	}

	parsedChannels := make([]parsedChannel, 0, len(pagedIDs.ChannelIDs))
	peerIDSet := make(map[string]struct{})

	for _, cid := range pagedIDs.ChannelIDs {
		meta, mErr := cmdMap[cid].Result()
		if mErr != nil || len(meta) == 0 {
			// Partial cache miss on underlying channel metadata Hash -> signal caller to backfill DB
			return nil, ErrMeCacheMiss
		}

		cUUID, pErr := uuid.Parse(cid)
		if pErr != nil {
			continue
		}

		ch := MeChannel{
			ID:      cUUID,
			Type:    meta["type"],
			Name:    meta["name"],
			IconURL: meta["icon_url"],
		}

		var peers []string
		if memberIDsStr, ok := meta["member_ids"]; ok && memberIDsStr != "" {
			for _, pid := range strings.Split(memberIDsStr, ",") {
				pid = strings.TrimSpace(pid)
				if pid != "" && pid != userID.String() {
					peers = append(peers, pid)
					peerIDSet[pid] = struct{}{}
				}
			}
		}

		parsedChannels = append(parsedChannels, parsedChannel{
			channel: ch,
			peerIDs: peers,
		})
	}

	// 3. Pipeline peer user profiles and presence
	peerMap := make(map[string]MeChannelPeer)
	if len(peerIDSet) > 0 {
		peerIDs := make([]string, 0, len(peerIDSet))
		for pid := range peerIDSet {
			peerIDs = append(peerIDs, pid)
		}

		profileCmds := make(map[string]*goredis.MapStringStringCmd, len(peerIDs))
		presenceCmds := make(map[string]*goredis.StringCmd, len(peerIDs))

		err = m.store.ExecPipelineFunc(ctx, func(pipe goredis.Pipeliner) error {
			for _, pid := range peerIDs {
				pUUID, pErr := uuid.Parse(pid)
				if pErr != nil {
					continue
				}
				profileCmds[pid] = pipe.HGetAll(ctx, userAggregateKey(pUUID))
				presenceCmds[pid] = pipe.Get(ctx, presenceKey(pUUID))
				pipe.Expire(ctx, userAggregateKey(pUUID), m.ttl)
			}
			return nil
		})
		if err != nil {
			return nil, redis.NewError(err, redis.ScopeMe)
		}

		for _, pid := range peerIDs {
			pUUID, pErr := uuid.Parse(pid)
			if pErr != nil {
				continue
			}

			profMap, _ := profileCmds[pid].Result()
			pres, _ := presenceCmds[pid].Result()

			if pres == "" {
				pres = "offline"
			}

			// If profile hash evicted, provide safe fallbacks instead of empty strings
			displayName := profMap["display_name"]
			if displayName == "" {
				displayName = profMap["username"]
			}

			peerMap[pid] = MeChannelPeer{
				ID:          pUUID,
				DisplayName: displayName,
				AvatarURL:   profMap["avatar_url"],
				Presence:    pres,
			}
		}
	}

	// 4. Assemble final channels slice with initialized peer slice
	channels := make([]MeChannel, 0, len(parsedChannels))
	for _, pc := range parsedChannels {
		ch := pc.channel
		ch.Peers = make([]MeChannelPeer, 0, len(pc.peerIDs))

		for _, pid := range pc.peerIDs {
			if peer, exists := peerMap[pid]; exists {
				ch.Peers = append(ch.Peers, peer)
			}
		}
		channels = append(channels, ch)
	}

	return &MeChannelResult{
		Channels:   channels,
		NextCursor: pagedIDs.NextCursor,
		HasMore:    pagedIDs.HasMore,
	}, nil
}

// TouchMeChannel updates or inserts a single channel's score in the user's ZSET
// and enforces the 500-channel ceiling by trimming the lowest-scored items.
func (m *Me) TouchMeChannel(ctx context.Context, userID uuid.UUID, channelID uuid.UUID, score int64) error {
	const maxChannels = 500
	key := meChannelsKey(userID)

	err := m.store.ExecPipelineFunc(ctx, func(pipe goredis.Pipeliner) error {
		pipe.ZAdd(ctx, key, goredis.Z{
			Score:  float64(score),
			Member: channelID.String(),
		})
		pipe.ZRemRangeByRank(ctx, key, 0, -maxChannels-1)
		pipe.Expire(ctx, key, m.ttl)
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopeMe)
	}

	return nil
}

// RemoveMeChannel removes a channel from the user's sidebar ZSET when they leave or delete a channel.
func (m *Me) RemoveMeChannel(ctx context.Context, userID uuid.UUID, channelID uuid.UUID) error {
	key := meChannelsKey(userID)

	err := m.store.ExecPipelineFunc(ctx, func(pipe goredis.Pipeliner) error {
		pipe.ZRem(ctx, key, channelID.String())
		pipe.Expire(ctx, key, m.ttl)
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopeMe)
	}

	return nil
}
