package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"

	"github.com/google/uuid"
)

type MemberRepository struct {
	store *db.Store
}

func NewMemberRepository(store *db.Store) *MemberRepository {
	return &MemberRepository{
		store: store.WithEntity(db.EntityChannelMember),
	}
}

func (r *MemberRepository) CreateBatch(ctx context.Context, members []*channel.Member) ([]*channel.Member, error) {
	if len(members) == 0 {
		return []*channel.Member{}, nil
	}

	type memberPayload struct {
		ChannelID         uuid.UUID  `json:"channel_id"`
		UserID            uuid.UUID  `json:"user_id"`
		LastReadMessageID *uuid.UUID `json:"last_read_message_id,omitempty"`
		CreatedAt         time.Time  `json:"created_at"`
		UpdatedAt         time.Time  `json:"updated_at"`
		LastReadMessageAt time.Time  `json:"last_read_message_at"`
		PinnedAt          *time.Time `json:"pinned_at,omitempty"`
		MutedUntil        *time.Time `json:"muted_until,omitempty"`
		MentionCount      int32      `json:"mention_count"`
		IsVisible         bool       `json:"is_visible"`
	}

	payloads := make([]memberPayload, len(members))
	for i, m := range members {
		payloads[i] = memberPayload{
			ChannelID:         m.ChannelID().UUID(),
			UserID:            m.UserID().UUID(),
			LastReadMessageID: m.LastReadMessageID().UUIDPtr(),
			CreatedAt:         m.CreatedAt().Time(),
			UpdatedAt:         m.UpdatedAt().Time(),
			LastReadMessageAt: m.LastReadMessageAt().Time(),
			PinnedAt:          m.PinnedAt().TimePtr(),
			MutedUntil:        m.MutedUntil().TimePtr(),
			MentionCount:      m.MentionCount(),
			IsVisible:         m.IsVisible(),
		}
	}

	jsonBytes, err := json.Marshal(payloads)
	if err != nil {
		return nil, errs.Internal("failed to marshal create batch payload").
			Meta("scope", db.EntityChannelMember.String()).
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
	row, err := r.store.ChannelMemberGet(ctx, db.ChannelMemberGetParams{
		ChannelID: db.ToUUID(channelID.UUID()),
		UserID:    db.ToUUID(userID.UUID()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return memberFromRow(row)
}

func (r *MemberRepository) Require(ctx context.Context, channelID, userID fields.ID) (*channel.Member, error) {
	member, err := r.Get(ctx, channelID, userID)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil, errs.PermissionDenied("You are not a member of this channel.")
		}
		return nil, err
	}
	return member, nil
}

func (r *MemberRepository) GetBatchByChannelIDs(
	ctx context.Context,
	channelIDs []fields.ID,
) (map[fields.ID][]*channel.Member, error) {
	if len(channelIDs) == 0 {
		return make(map[fields.ID][]*channel.Member), nil
	}

	result := make(map[fields.ID][]*channel.Member, len(channelIDs))
	uuids := make([]uuid.UUID, len(channelIDs))

	for i, id := range channelIDs {
		uuidVal := id.UUID()
		uuids[i] = uuidVal
		result[id] = []*channel.Member{}
	}

	rows, err := r.store.ChannelMemberGetBatchByChannelIDs(ctx, db.ToUUIDs(uuids))
	if err != nil {
		return nil, r.store.Err(err)
	}

	for _, row := range rows {
		m, err := memberFromRow(row)
		if err != nil {
			return nil, err
		}

		cid := m.ChannelID()
		result[cid] = append(result[cid], m)
	}

	return result, nil
}

func (r *MemberRepository) GetBatchByChannelID(
	ctx context.Context,
	channelID fields.ID,
) ([]*channel.Member, error) {
	memberMap, err := r.GetBatchByChannelIDs(ctx, []fields.ID{channelID})
	if err != nil {
		return nil, err
	}

	members, ok := memberMap[channelID]
	if !ok || len(members) == 0 {
		return nil, errs.NotFound("entity not found")
	}

	return members, nil
}

func (r *MemberRepository) ListVisibleByUserID(ctx context.Context, userID fields.ID, limit int) ([]*channel.Member, error) {
	rows, err := r.store.ChannelMemberListVisibleByUserID(ctx, db.ChannelMemberListVisibleByUserIDParams{
		UserID:   db.ToUUID(userID.UUID()),
		LimitVal: int32(limit),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	members := make([]*channel.Member, 0, len(rows))
	for _, row := range rows {
		m, err := memberFromRow(row)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	return members, nil
}

func (r *MemberRepository) CountByChannelID(ctx context.Context, channelID fields.ID) (int, error) {
	count, err := r.store.ChannelMemberCountByChannelID(ctx, db.ToUUID(channelID.UUID()))
	if err != nil {
		return 0, r.store.Err(err)
	}

	return int(count), nil
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
	mentionCount *int,
) (*channel.Member, error) {
	row, err := r.store.ChannelMemberUpdateLastReadMessage(ctx, db.ChannelMemberUpdateLastReadMessageParams{
		ChannelID:         db.ToUUID(channelID.UUID()),
		UserID:            db.ToUUID(userID.UUID()),
		LastReadMessageID: db.ToUUIDPtr(lastReadMessageID.UUIDPtr()),
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

func (r *MemberRepository) ClearBatchLastReadMessageByChannelID(
	ctx context.Context,
	channelID fields.ID,
	lastReadAt, updatedAt fields.Timestamp,
) ([]*channel.Member, error) {
	rows, err := r.store.ChannelMemberClearBatchLastReadMessageByChannelID(ctx, db.ChannelMemberClearBatchLastReadMessageByChannelIDParams{
		ChannelID:         db.ToUUID(channelID.UUID()),
		LastReadMessageAt: db.ToTimestamptz(lastReadAt.Time()),
		UpdatedAt:         db.ToTimestamptz(updatedAt.Time()),
	})
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

	lastReadMessageID, err := fields.ParseID(db.FromUUID[uuid.UUID](row.LastReadMessageID))
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
