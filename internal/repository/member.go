package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

type MemberCache interface {
	Get(ctx context.Context, channelID fields.ID, userID fields.ID) (*channel.Member, error)
	GetBatchByChannelIDs(ctx context.Context, channelIDs []fields.ID) (found map[fields.ID][]*channel.Member, missingChannelIDs []fields.ID, err error)
	GetVisibleByUserID(ctx context.Context, userID fields.ID, limit int32) ([]*channel.Member, []fields.ID, error)
	Set(ctx context.Context, ch *channel.Member) error
	SetBatch(ctx context.Context, members []*channel.Member) error
}

type MemberRepository struct {
	store *db.Store
	cache MemberCache
	sf    singleflight.Group
}

func NewMemberRepository(store *db.Store, cache MemberCache) *MemberRepository {
	return &MemberRepository{
		store: store.WithEntity(db.EntityChannelMember),
		cache: cache,
	}
}

func (r *MemberRepository) CreateBatch(ctx context.Context, members []*channel.Member) ([]*channel.Member, error) {
	type memberPayload struct {
		ChannelID         uuid.UUID  `json:"channel_id"`
		UserID            uuid.UUID  `json:"user_id"`
		LastReadMessageID *uuid.UUID `json:"last_read_message_id,omitempty"`
		CreatedAt         string     `json:"created_at"`
		UpdatedAt         string     `json:"updated_at"`
		LastReadMessageAt *string    `json:"last_read_message_at,omitempty"`
		PinnedAt          *string    `json:"pinned_at,omitempty"`
		MutedUntil        *string    `json:"muted_until,omitempty"`
		MentionCount      int32      `json:"mention_count"`
		IsVisible         bool       `json:"is_visible"`
	}

	payloads := make([]memberPayload, len(members))
	for i, m := range members {
		payloads[i] = memberPayload{
			ChannelID:         m.ChannelID().UUID(),
			UserID:            m.UserID().UUID(),
			LastReadMessageID: m.LastReadMessageID().UUIDPtr(),
			CreatedAt:         m.CreatedAt().String(),
			UpdatedAt:         m.UpdatedAt().String(),
			LastReadMessageAt: m.LastReadMessageAt().StringPtr(),
			PinnedAt:          m.PinnedAt().StringPtr(),
			MutedUntil:        m.MutedUntil().StringPtr(),
			MentionCount:      m.MentionCount(),
			IsVisible:         m.IsVisible(),
		}
	}

	jsonBytes, err := json.Marshal(payloads)
	if err != nil {
		return nil, errs.Internal("failed to marshal create batch payload").
			Meta("scope", redis.ScopeMember.String()).
			Wrap(err)
	}

	rows, err := r.store.ChannelMemberCreateBatch(ctx, jsonBytes)
	if err != nil {
		return nil, r.store.Err(err)
	}

	result := make([]*channel.Member, len(rows))
	for i, row := range rows {
		m, err := memberFromRow(row)
		if err != nil {
			return nil, err
		}
		result[i] = m
	}

	return result, nil
}

func (r *MemberRepository) Get(ctx context.Context, channelID, userID fields.ID) (*channel.Member, error) {
	member, err := r.cache.Get(ctx, channelID, userID)
	if err == nil && member != nil {
		return member, nil
	}

	if err != nil {
		slog.WarnContext(ctx, "cache read failed, falling back to database",
			"channel_id", channelID.String(),
			"user_id", userID.String(),
			"error", err,
			"scope", redis.ScopeMember,
		)
	}

	sfKey := "member:" + channelID.String() + ":" + userID.String()
	sfCtx := context.WithoutCancel(ctx)

	val, err, _ := r.sf.Do(sfKey, func() (any, error) {
		row, err := r.store.ChannelMemberGet(sfCtx, db.ChannelMemberGetParams{
			ChannelID: db.ToUUID(channelID.UUID()),
			UserID:    db.ToUUID(userID.UUID()),
		})
		if err != nil {
			return nil, r.store.Err(err)
		}

		dbMember, err := memberFromRow(row)
		if err != nil {
			return nil, err
		}

		if cacheErr := r.cache.Set(sfCtx, dbMember); cacheErr != nil {
			slog.WarnContext(sfCtx, "failed to backfill cache",
				"channel_id", channelID.String(),
				"user_id", userID.String(),
				"error", cacheErr,
				"scope", redis.ScopeMember,
			)
		}

		return dbMember, nil
	})
	if err != nil {
		return nil, err
	}

	return val.(*channel.Member), nil
}

func (r *MemberRepository) GetBatchByChannelIDs(
	ctx context.Context,
	channelIDs []fields.ID,
) (map[fields.ID][]*channel.Member, error) {
	result, missingChannelIDs, err := r.cache.GetBatchByChannelIDs(ctx, channelIDs)
	if err != nil {
		slog.WarnContext(ctx, "cache batch read failed for channel members, falling back to database",
			"count", len(channelIDs),
			"error", err,
			"scope", redis.ScopeMember,
		)
		result = make(map[fields.ID][]*channel.Member)
		missingChannelIDs = channelIDs
	}

	if len(missingChannelIDs) == 0 {
		return result, nil
	}

	uuids := make([]uuid.UUID, len(missingChannelIDs))
	for i, id := range missingChannelIDs {
		uuids[i] = id.UUID()
	}

	rows, err := r.store.ChannelMemberGetBatchByChannelIDs(ctx, db.ToUUIDs(uuids))
	if err != nil {
		return nil, r.store.Err(err)
	}

	dbMembersByChannel := make(map[fields.ID][]*channel.Member)
	allDBMembers := make([]*channel.Member, 0, len(rows))

	for _, row := range rows {
		m, err := memberFromRow(row)
		if err != nil {
			return nil, err
		}

		cid := m.ChannelID()
		dbMembersByChannel[cid] = append(dbMembersByChannel[cid], m)
		allDBMembers = append(allDBMembers, m)
	}

	for _, cid := range missingChannelIDs {
		members := dbMembersByChannel[cid]
		if members == nil {
			members = []*channel.Member{}
		}
		result[cid] = members
	}

	if len(allDBMembers) > 0 {
		if cacheErr := r.cache.SetBatch(ctx, allDBMembers); cacheErr != nil {
			slog.WarnContext(ctx, "failed to backfill member cache",
				"count", len(allDBMembers),
				"error", cacheErr,
				"scope", redis.ScopeMember,
			)
		}
	}

	return result, nil
}

func (r *MemberRepository) ListVisibleByUserID(ctx context.Context, userID fields.ID, limit int32) ([]*channel.Member, error) {
	members, _, err := r.cache.GetVisibleByUserID(ctx, userID, limit)
	if err == nil && len(members) > 0 {
		return members, nil
	}
	if err != nil {
		slog.WarnContext(ctx, "cache read failed for visible members, falling back to database",
			"user_id", userID,
			"error", err,
			"scope", redis.ScopeMember,
		)
	}

	rows, err := r.store.ChannelMemberListVisibleByUserID(ctx, db.ChannelMemberListVisibleByUserIDParams{
		UserID:   db.ToUUID(userID.UUID()),
		LimitVal: limit,
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	members = make([]*channel.Member, 0, len(rows))
	for _, row := range rows {
		m, err := memberFromRow(row)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	if len(members) > 0 {
		if cacheErr := r.cache.SetBatch(ctx, members); cacheErr != nil {
			slog.WarnContext(ctx, "failed to backfill cache",
				"user_id", userID,
				"error", cacheErr,
				"scope", redis.ScopeMember,
			)
		}
	}

	return members, nil
}

func (r *MemberRepository) CountByChannel(ctx context.Context, channelID fields.ID) (int64, error) {
	count, err := r.store.ChannelMemberCountByChannel(ctx, db.ToUUID(channelID.UUID()))
	if err != nil {
		return 0, r.store.Err(err)
	}

	return count, nil
}

func (r *MemberRepository) UpdateIsVisible(
	ctx context.Context,
	channelID, userID fields.ID,
	isVisible bool,
	updatedAt fields.Timestamp,
) (*channel.Member, error) {
	row, err := r.store.ChannelMemberUpdateIsVisible(ctx, db.ChannelMemberUpdateIsVisibleParams{
		ChannelID: db.ToUUID(channelID.UUID()),
		UserID:    db.ToUUID(userID.UUID()),
		IsVisible: isVisible,
		UpdatedAt: db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return memberFromRow(row)
}

func (r *MemberRepository) UpdateLastReadMessage(
	ctx context.Context,
	channelID, userID, lastReadMessageID fields.ID,
	lastReadMessageAt, updatedAt fields.Timestamp,
	mentionCount *int32,
) (*channel.Member, error) {
	row, err := r.store.ChannelMemberUpdateLastReadMessage(ctx, db.ChannelMemberUpdateLastReadMessageParams{
		ChannelID:         db.ToUUID(channelID.UUID()),
		UserID:            db.ToUUID(userID.UUID()),
		LastReadMessageID: db.ToUUID(lastReadMessageID.UUID()),
		LastReadMessageAt: db.ToTimestamptz(lastReadMessageAt.Time()),
		MentionCount:      db.ToInt4Ptr(mentionCount),
		UpdatedAt:         db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return memberFromRow(row)
}

func (r *MemberRepository) UpdatePinnedAt(
	ctx context.Context,
	channelID, userID fields.ID,
	pinnedAt, updatedAt fields.Timestamp,
) (*channel.Member, error) {
	row, err := r.store.ChannelMemberUpdatePinnedAt(ctx, db.ChannelMemberUpdatePinnedAtParams{
		ChannelID: db.ToUUID(channelID.UUID()),
		UserID:    db.ToUUID(userID.UUID()),
		PinnedAt:  db.ToTimestamptzPtr(pinnedAt.TimePtr()),
		UpdatedAt: db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return memberFromRow(row)
}

func (r *MemberRepository) UpdateMutedUntil(
	ctx context.Context,
	channelID, userID fields.ID,
	mutedUntil, updatedAt fields.Timestamp,
) (*channel.Member, error) {
	row, err := r.store.ChannelMemberUpdateMutedUntil(ctx, db.ChannelMemberUpdateMutedUntilParams{
		ChannelID:  db.ToUUID(channelID.UUID()),
		UserID:     db.ToUUID(userID.UUID()),
		MutedUntil: db.ToTimestamptzPtr(mutedUntil.TimePtr()),
		UpdatedAt:  db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return memberFromRow(row)
}

func (r *MemberRepository) IncrementPeersMentionCountByChannelID(
	ctx context.Context,
	channelID, userID fields.ID,
	updatedAt fields.Timestamp,
) error {
	err := r.store.ChannelMemberIncrementPeersMentionCountByChannelID(ctx, db.ChannelMemberIncrementPeersMentionCountByChannelIDParams{
		ChannelID: db.ToUUID(channelID.UUID()),
		UserID:    db.ToUUID(userID.UUID()),
		UpdatedAt: db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

func (r *MemberRepository) Delete(ctx context.Context, channelID, userID fields.ID) error {
	err := r.store.ChannelMemberDelete(ctx, db.ChannelMemberDeleteParams{
		ChannelID: db.ToUUID(channelID.UUID()),
		UserID:    db.ToUUID(userID.UUID()),
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

func memberFromRow(row db.ChannelMember) (*channel.Member, error) {
	channelID := db.FromUUID[uuid.UUID](row.ChannelID)
	userID := db.FromUUID[uuid.UUID](row.UserID)
	compositeKey := fmt.Sprintf("%s:%s", channelID.String(), userID.String())

	mapErr := func(msg, key string, val any, err error) *errs.Error {
		return errs.Internal(msg).
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta(key, fmt.Sprintf("%v", val)).
			Resource("ChannelMember", compositeKey, "", "database row mapping")
	}

	parsedChannelID, err := fields.ParseRequiredID("channel_id", channelID)
	if err != nil {
		return nil, mapErr("failed to parse channel id from database", "channel_id", channelID.String(), err)
	}

	parsedUserID, err := fields.ParseRequiredID("user_id", userID)
	if err != nil {
		return nil, mapErr("failed to parse user id from database", "user_id", userID.String(), err)
	}

	lastReadMessageID, err := fields.ParseID("last_read_message_id", db.FromUUID[uuid.UUID](row.LastReadMessageID))
	if err != nil {
		return nil, mapErr("failed to parse last read message id from database", "last_read_message_id", row.LastReadMessageID, err)
	}

	lastReadMessageAt := fields.NewTimestamp(db.FromTimestamptz(row.LastReadMessageAt))
	pinnedAt := fields.NewTimestamp(db.FromTimestamptz(row.PinnedAt))
	mutedUntil := fields.NewTimestamp(db.FromTimestamptz(row.MutedUntil))
	createdAt := fields.NewTimestamp(db.FromTimestamptz(row.CreatedAt))
	updatedAt := fields.NewTimestamp(db.FromTimestamptz(row.UpdatedAt))

	return channel.ParseMember(
		parsedChannelID,
		parsedUserID,
		lastReadMessageID,
		lastReadMessageAt,
		pinnedAt,
		mutedUntil,
		row.MentionCount,
		row.IsVisible,
		createdAt,
		updatedAt,
	), nil
}
