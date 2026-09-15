package repository

import (
	"context"
	"encoding/json"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"

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
		LastReadMessageAt *time.Time `json:"last_read_message_at,omitempty"`
		PinnedAt          *time.Time `json:"pinned_at,omitempty"`
		MutedUntil        *time.Time `json:"muted_until,omitempty"`
		MentionCount      int32      `json:"mention_count"`
		IsVisible         bool       `json:"is_visible"`
	}

	payloads := make([]memberPayload, len(members))
	for i, m := range members {
		payloads[i] = memberPayload{
			ChannelID:         m.ChannelID,
			UserID:            m.UserID,
			LastReadMessageID: m.LastReadMessageID,
			CreatedAt:         m.CreatedAt,
			UpdatedAt:         m.UpdatedAt,
			LastReadMessageAt: m.LastReadMessageAt,
			PinnedAt:          m.PinnedAt,
			MutedUntil:        m.MutedUntil,
			MentionCount:      int32(m.MentionCount),
			IsVisible:         m.IsVisible,
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
		result[i] = memberFromRow(row)
	}

	return result, nil
}

func (r *MemberRepository) Get(ctx context.Context, channelID, userID uuid.UUID) (*channel.Member, error) {
	row, err := r.store.ChannelMemberGet(ctx, db.ChannelMemberGetParams{
		ChannelID: db.ToUUID(channelID),
		UserID:    db.ToUUID(userID),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return memberFromRow(row), nil
}

func (r *MemberRepository) GetBatchByChannelIDs(
	ctx context.Context,
	channelIDs []uuid.UUID,
) (map[uuid.UUID][]*channel.Member, error) {
	if len(channelIDs) == 0 {
		return make(map[uuid.UUID][]*channel.Member), nil
	}

	result := make(map[uuid.UUID][]*channel.Member, len(channelIDs))
	uuids := make([]uuid.UUID, len(channelIDs))

	for i, id := range channelIDs {
		uuidVal := id
		uuids[i] = uuidVal
		result[id] = []*channel.Member{}
	}

	rows, err := r.store.ChannelMemberGetBatchByChannelIDs(ctx, db.ToUUIDs(uuids))
	if err != nil {
		return nil, r.store.Err(err)
	}

	for _, row := range rows {
		m := memberFromRow(row)
		cid := m.ChannelID
		result[cid] = append(result[cid], m)
	}

	return result, nil
}

func (r *MemberRepository) GetBatchByChannelID(
	ctx context.Context,
	channelID uuid.UUID,
) ([]*channel.Member, error) {
	memberMap, err := r.GetBatchByChannelIDs(ctx, []uuid.UUID{channelID})
	if err != nil {
		return nil, err
	}

	members, ok := memberMap[channelID]
	if !ok || len(members) == 0 {
		return nil, errs.NotFound("entity not found")
	}

	return members, nil
}

func (r *MemberRepository) ListVisibleByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*channel.Member, error) {
	rows, err := r.store.ChannelMemberListVisibleByUserID(ctx, db.ChannelMemberListVisibleByUserIDParams{
		UserID:   db.ToUUID(userID),
		LimitVal: int32(limit),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	members := make([]*channel.Member, 0, len(rows))
	for _, row := range rows {
		members = append(members, memberFromRow(row))
	}

	return members, nil
}

func (r *MemberRepository) CountByChannelID(ctx context.Context, channelID uuid.UUID) (int, error) {
	count, err := r.store.ChannelMemberCountByChannelID(ctx, db.ToUUID(channelID))
	if err != nil {
		return 0, r.store.Err(err)
	}

	return int(count), nil
}

func (r *MemberRepository) UpdateIsVisible(
	ctx context.Context,
	channelID, userID uuid.UUID,
	isVisible bool,
	updatedAt time.Time,
) (*channel.Member, error) {
	row, err := r.store.ChannelMemberUpdateIsVisible(ctx, db.ChannelMemberUpdateIsVisibleParams{
		ChannelID: db.ToUUID(channelID),
		UserID:    db.ToUUID(userID),
		IsVisible: isVisible,
		UpdatedAt: db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return memberFromRow(row), nil
}

func (r *MemberRepository) UpdateLastReadMessage(
	ctx context.Context,
	channelID, userID uuid.UUID,
	lastReadMessageID *uuid.UUID,
	lastReadMessageAt, updatedAt time.Time,
	mentionCount *int,
) (*channel.Member, error) {
	row, err := r.store.ChannelMemberUpdateLastReadMessage(ctx, db.ChannelMemberUpdateLastReadMessageParams{
		ChannelID:         db.ToUUID(channelID),
		UserID:            db.ToUUID(userID),
		LastReadMessageID: db.ToUUIDPtr(lastReadMessageID),
		LastReadMessageAt: db.ToTimestamptz(lastReadMessageAt),
		MentionCount:      db.ToInt4Ptr(mentionCount),
		UpdatedAt:         db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return memberFromRow(row), nil
}

func (r *MemberRepository) UpdatePinnedAt(
	ctx context.Context,
	channelID, userID uuid.UUID,
	pinnedAt *time.Time,
	updatedAt time.Time,
) (*channel.Member, error) {
	row, err := r.store.ChannelMemberUpdatePinnedAt(ctx, db.ChannelMemberUpdatePinnedAtParams{
		ChannelID: db.ToUUID(channelID),
		UserID:    db.ToUUID(userID),
		PinnedAt:  db.ToTimestamptzPtr(pinnedAt),
		UpdatedAt: db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return memberFromRow(row), nil
}

func (r *MemberRepository) UpdateMutedUntil(
	ctx context.Context,
	channelID, userID uuid.UUID,
	mutedUntil *time.Time,
	updatedAt time.Time,
) (*channel.Member, error) {
	row, err := r.store.ChannelMemberUpdateMutedUntil(ctx, db.ChannelMemberUpdateMutedUntilParams{
		ChannelID:  db.ToUUID(channelID),
		UserID:     db.ToUUID(userID),
		MutedUntil: db.ToTimestamptzPtr(mutedUntil),
		UpdatedAt:  db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return memberFromRow(row), nil
}

func (r *MemberRepository) IncrementPeersMentionCountByChannelID(
	ctx context.Context,
	channelID, userID uuid.UUID,
	incrementAmount int,
	updatedAt time.Time,
) error {
	err := r.store.ChannelMemberIncrementPeersMentionCountByChannelID(ctx, db.ChannelMemberIncrementPeersMentionCountByChannelIDParams{
		IncrementAmount: int32(incrementAmount),
		ChannelID:       db.ToUUID(channelID),
		UserID:          db.ToUUID(userID),
		UpdatedAt:       db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

func (r *MemberRepository) Delete(ctx context.Context, channelID, userID uuid.UUID) error {
	err := r.store.ChannelMemberDelete(ctx, db.ChannelMemberDeleteParams{
		ChannelID: db.ToUUID(channelID),
		UserID:    db.ToUUID(userID),
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

func memberFromRow(row db.ChannelMember) *channel.Member {
	return channel.ReconstituteMember(
		db.FromUUID[uuid.UUID](row.ChannelID),
		db.FromUUID[uuid.UUID](row.UserID),
		db.FromUUIDPtr[uuid.UUID](row.LastReadMessageID),
		db.FromTimestamptzPtr(row.LastReadMessageAt),
		db.FromTimestamptzPtr(row.PinnedAt),
		db.FromTimestamptzPtr(row.MutedUntil),
		int(row.MentionCount),
		row.IsVisible,
		db.FromTimestamptz(row.CreatedAt),
		db.FromTimestamptz(row.UpdatedAt),
	)
}
