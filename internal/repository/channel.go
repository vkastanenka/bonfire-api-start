package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Channel struct {
	store db.Querier
}

func NewChannel(store db.Querier) *Channel {
	return &Channel{store: store}
}

// ============================================================================
// CHANNELS
// ============================================================================

func (r *Channel) Create(ctx context.Context, ch *channel.Channel) (*channel.Channel, error) {
	row, err := r.store.ChannelCreate(ctx, db.ChannelCreateParams{
		ID:        db.UUID(uuid.UUID(ch.ID())),
		Type:      int16(ch.Type()),
		Name:      db.StringerPtr(ch.Name()),
		IconUrl:   db.StringerPtr(ch.IconURL()),
		CreatedAt: db.Timestamptz(ch.CreatedAt()),
		UpdatedAt: db.Timestamptz(ch.UpdatedAt()),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityChannel)
	}

	return channelFromRow(row)
}

func (r *Channel) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.ChannelDelete(ctx, db.UUID(id))
	if err != nil {
		return db.NewError(err, db.EntityChannel)
	}

	return nil
}

func (r *Channel) Get(ctx context.Context, id uuid.UUID) (*channel.Channel, error) {
	row, err := r.store.ChannelGet(ctx, db.UUID(id))
	if err != nil {
		return nil, db.NewError(err, db.EntityChannel)
	}

	return channelFromRow(row)
}

func (r *Channel) GetForMember(ctx context.Context, channelID, memberID uuid.UUID) (*channel.Channel, error) {
	row, err := r.store.ChannelGetForMember(ctx, db.ChannelGetForMemberParams{
		ChannelID: db.UUID(channelID),
		UserID:    db.UUID(memberID),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityChannel)
	}

	return channelFromRow(row)
}

func (r *Channel) GetForMemberUpdate(ctx context.Context, channelID, memberID uuid.UUID) (*channel.Channel, error) {
	row, err := r.store.ChannelGetForMemberUpdate(ctx, db.ChannelGetForMemberUpdateParams{
		ChannelID: db.UUID(channelID),
		UserID:    db.UUID(memberID),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityChannel)
	}

	return channelFromRow(row)
}

func (r *Channel) HasMessagesAfter(ctx context.Context, channelID uuid.UUID, createdAt time.Time, id uuid.UUID) (bool, error) {
	exists, err := r.store.ChannelHasMessagesAfter(ctx, db.ChannelHasMessagesAfterParams{
		ChannelID: db.UUID(channelID),
		CreatedAt: db.Timestamptz(createdAt),
		MessageID: db.UUID(id),
	})
	if err != nil {
		return false, db.NewError(err, db.EntityMessage)
	}

	return exists, nil
}

func (r *Channel) HasMessagesBefore(ctx context.Context, channelID uuid.UUID, createdAt time.Time, id uuid.UUID) (bool, error) {
	exists, err := r.store.ChannelHasMessagesBefore(ctx, db.ChannelHasMessagesBeforeParams{
		ChannelID: db.UUID(channelID),
		CreatedAt: db.Timestamptz(createdAt),
		MessageID: db.UUID(id),
	})
	if err != nil {
		return false, db.NewError(err, db.EntityMessage)
	}

	return exists, nil
}

func (r *Channel) Update(ctx context.Context, ch *channel.Channel) (*channel.Channel, error) {
	row, err := r.store.ChannelUpdate(ctx, db.ChannelUpdateParams{
		ID:        db.UUID(uuid.UUID(ch.ID())),
		Name:      db.StringerPtr(ch.Name()),
		IconUrl:   db.StringerPtr(ch.IconURL()),
		UpdatedAt: db.Timestamptz(ch.UpdatedAt()),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityChannel)
	}

	return channelFromRow(row)
}

func (r *Channel) UpdateLastMessage(ctx context.Context, ch *channel.Channel) (*channel.Channel, error) {
	var rawMsgID *uuid.UUID
	if msgID := ch.LastMessageID(); msgID != nil {
		u := msgID.UUID()
		rawMsgID = &u
	}

	row, err := r.store.ChannelUpdateLastMessage(ctx, db.ChannelUpdateLastMessageParams{
		ChannelID:     db.UUID(ch.ID().UUID()),
		LastMessageID: db.UUIDPtr(rawMsgID),
		UpdatedAt:     db.Timestamptz(ch.UpdatedAt()),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityChannel)
	}

	return channelFromRow(row)
}

// ============================================================================
// MEMBERS
// ============================================================================

func (r *Channel) MemberAddBatch(ctx context.Context, members []*channel.Member) error {
	if len(members) == 0 {
		return nil
	}

	channelID := members[0].ChannelID()

	userIDs := make([]pgtype.UUID, len(members))
	createdAts := make([]pgtype.Timestamptz, len(members))
	updatedAts := make([]pgtype.Timestamptz, len(members))
	lastReadAts := make([]pgtype.Timestamptz, len(members))
	mentionCounts := make([]int32, len(members))
	lastReadMessageIDs := make([]pgtype.UUID, len(members))

	for i, m := range members {
		userIDs[i] = db.UUID(m.UserID().UUID())
		createdAts[i] = db.Timestamptz(m.CreatedAt())
		updatedAts[i] = db.Timestamptz(m.UpdatedAt())
		lastReadAts[i] = db.Timestamptz(m.LastReadAt())
		mentionCounts[i] = m.MentionCount()

		var rawMsgID *uuid.UUID
		if msgID := m.LastReadMessageID(); msgID != nil {
			u := msgID.UUID()
			rawMsgID = &u
		}
		lastReadMessageIDs[i] = db.UUIDPtr(rawMsgID)
	}

	err := r.store.ChannelMemberAddBatch(ctx, db.ChannelMemberAddBatchParams{
		ChannelID:          db.UUID(channelID.UUID()),
		UserIds:            userIDs,
		CreatedAts:         createdAts,
		UpdatedAts:         updatedAts,
		LastReadAts:        lastReadAts,
		MentionCounts:      mentionCounts,
		LastReadMessageIds: lastReadMessageIDs,
	})
	if err != nil {
		return db.NewError(err, db.EntityChannelMember)
	}

	return nil
}

func (r *Channel) MemberCount(ctx context.Context, channelID uuid.UUID) (int32, error) {
	count, err := r.store.ChannelMemberCount(ctx, db.UUID(channelID))
	if err != nil {
		return 0, db.NewError(err, db.EntityChannelMember)
	}

	return count, nil
}

func (r *Channel) MemberGet(ctx context.Context, channelID, userID uuid.UUID) (*channel.Member, error) {
	row, err := r.store.ChannelMemberGet(ctx, db.ChannelMemberGetParams{
		ChannelID: db.UUID(channelID),
		UserID:    db.UUID(userID),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityChannelMember)
	}

	return memberFromRow(row)
}

func (r *Channel) MemberGetUnreadCount(ctx context.Context, channelID, userID uuid.UUID) (int32, error) {
	count, err := r.store.ChannelMemberGetUnreadCount(ctx, db.ChannelMemberGetUnreadCountParams{
		ChannelID: db.UUID(channelID),
		UserID:    db.UUID(userID),
	})
	if err != nil {
		return 0, db.NewError(err, db.EntityChannelMember)
	}

	return count, nil
}

func (r *Channel) MemberIncrementMentionCountBatch(ctx context.Context, channelID uuid.UUID, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}

	dbUUIDs := make([]pgtype.UUID, len(userIDs))
	for i, id := range userIDs {
		dbUUIDs[i] = db.UUID(id)
	}

	err := r.store.ChannelMemberIncrementMentionCountBatch(ctx, db.ChannelMemberIncrementMentionCountBatchParams{
		ChannelID: db.UUID(channelID),
		UserIds:   dbUUIDs,
	})
	if err != nil {
		return db.NewError(err, db.EntityChannelMember)
	}

	return nil
}

func (r *Channel) MemberListByChannel(ctx context.Context, channelID uuid.UUID) ([]*channel.Member, error) {
	rows, err := r.store.ChannelMemberListByChannel(ctx, db.UUID(channelID))
	if err != nil {
		return nil, db.NewError(err, db.EntityChannelMember)
	}

	members := make([]*channel.Member, 0, len(rows))
	for _, row := range rows {
		member, err := memberFromRow(row)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, nil
}

func (r *Channel) MemberListItemsByChannel(
	ctx context.Context,
	channelID uuid.UUID,
	cursorCreatedAt *time.Time,
	cursorUserID *uuid.UUID,
	limit int32,
) ([]*channel.MemberListItem, error) {
	rows, err := r.store.ChannelMemberListItemsByChannel(ctx, db.ChannelMemberListItemsByChannelParams{
		ChannelID:       db.UUID(channelID),
		CursorCreatedAt: db.TimestamptzPtr(cursorCreatedAt),
		CursorUserID:    db.UUIDPtr(cursorUserID),
		LimitVal:        limit,
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityChannelMember)
	}

	members := make([]*channel.MemberListItem, 0, len(rows))
	for _, row := range rows {
		member, err := memberListItemFromRow(row)
		if err != nil {
			return nil, db.NewError(err, db.EntityChannelMember)
		}
		members = append(members, member)
	}

	return members, nil
}

func (r *Channel) MemberRemove(ctx context.Context, channelID, userID uuid.UUID) error {
	err := r.store.ChannelMemberRemove(ctx, db.ChannelMemberRemoveParams{
		ChannelID: db.UUID(channelID),
		UserID:    db.UUID(userID),
	})
	if err != nil {
		return db.NewError(err, db.EntityChannelMember)
	}

	return nil
}

func (r *Channel) MemberResetMentionCount(ctx context.Context, channelID, userID uuid.UUID) error {
	err := r.store.ChannelMemberResetMentionCount(ctx, db.ChannelMemberResetMentionCountParams{
		ChannelID: db.UUID(channelID),
		UserID:    db.UUID(userID),
	})
	if err != nil {
		return db.NewError(err, db.EntityChannelMember)
	}

	return nil
}

func (r *Channel) MemberUpdateLastRead(ctx context.Context, channelID, userID uuid.UUID, messageID *uuid.UUID, lastReadAt time.Time) error {
	err := r.store.ChannelMemberUpdateLastRead(ctx, db.ChannelMemberUpdateLastReadParams{
		ChannelID:         db.UUID(channelID),
		UserID:            db.UUID(userID),
		LastReadMessageID: db.UUIDPtr(messageID),
		LastReadAt:        db.Timestamptz(lastReadAt),
	})
	if err != nil {
		return db.NewError(err, db.EntityChannelMember)
	}

	return nil
}

func (r *Channel) IsMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
	_, err := r.store.ChannelMemberGet(ctx, db.ChannelMemberGetParams{
		ChannelID: db.UUID(channelID),
		UserID:    db.UUID(userID),
	})
	if err != nil {
		if db.IsNotFoundError(err) {
			return false, nil
		}
		return false, db.NewError(err, db.EntityChannelMember)
	}
	return true, nil
}

// ============================================================================
// MESSAGES
// ============================================================================

func (r *Channel) MessageCreate(ctx context.Context, msg *channel.Message) error {
	var authorID *uuid.UUID
	if id := msg.AuthorID(); id != nil {
		u := id.UUID()
		authorID = &u
	}

	var replyToMsgID *uuid.UUID
	if id := msg.ReplyToMessageID(); id != nil {
		u := id.UUID()
		replyToMsgID = &u
	}

	_, err := r.store.MessageCreate(ctx, db.MessageCreateParams{
		ID:               db.UUID(msg.ID().UUID()),
		ChannelID:        db.UUID(msg.ChannelID().UUID()),
		AuthorID:         db.UUIDPtr(authorID),
		ReplyToMessageID: db.UUIDPtr(replyToMsgID),
		Content:          db.Text(msg.Content().String()),
		IsPinned:         msg.IsPinned(),
		CreatedAt:        db.Timestamptz(msg.CreatedAt()),
		UpdatedAt:        db.Timestamptz(msg.UpdatedAt()),
	})
	if err != nil {
		return db.NewError(err, db.EntityMessage)
	}

	return nil
}

func (r *Channel) MessageDelete(ctx context.Context, id uuid.UUID) error {
	err := r.store.MessageDelete(ctx, db.UUID(id))
	if err != nil {
		return db.NewError(err, db.EntityMessage)
	}

	return nil
}

func (r *Channel) MessageGet(
	ctx context.Context,
	id uuid.UUID,
	userID *uuid.UUID,
) (*channel.MessageAggregate, error) {
	row, err := r.store.MessageGetAggregate(ctx, db.UUID(id))
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	return messageAggregateFromRow(row, userID)
}

func (r *Channel) MessageGetFirstUnread(ctx context.Context, channelID, userID uuid.UUID) (*channel.Message, error) {
	row, err := r.store.MessageGetFirstUnread(ctx, db.MessageGetFirstUnreadParams{
		ChannelID: db.UUID(channelID),
		UserID:    db.UUID(userID),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	return messageFromRow(row)
}

func (r *Channel) MessageGetLatest(ctx context.Context, channelID uuid.UUID) (*channel.Message, error) {
	row, err := r.store.MessageGetLatest(ctx, db.UUID(channelID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, db.NewError(err, db.EntityMessage)
	}

	return messageFromRow(row)
}

// func (r *Channel) MessageListByChannelAfter(
// 	ctx context.Context,
// 	channelID uuid.UUID,
// 	cursorCreatedAt *time.Time,
// 	cursorID *uuid.UUID,
// 	limit int32,
// ) ([]channel.Message, error) {
// 	rows, err := r.store.MessageListByChannelAfter(ctx, db.MessageListByChannelAfterParams{
// 		ChannelID:       db.UUID(channelID),
// 		CursorCreatedAt: db.TimestamptzPtr(cursorCreatedAt),
// 		CursorID:        db.UUIDPtr(cursorID),
// 		ResultLimit:     limit,
// 	})
// 	if err != nil {
// 		return nil, db.NewError(err, db.EntityMessage)
// 	}

// 	return messagesFromRows(rows)
// }

// func (r *Channel) MessageListByChannelAround(
// 	ctx context.Context,
// 	channelID uuid.UUID,
// 	cursorCreatedAt time.Time,
// 	cursorID uuid.UUID,
// 	halfLimit int32,
// ) ([]channel.Message, error) {
// 	rows, err := r.store.MessageListByChannelAround(ctx, db.MessageListByChannelAroundParams{
// 		ChannelID:       db.UUID(channelID),
// 		CursorCreatedAt: db.Timestamptz(cursorCreatedAt),
// 		CursorID:        db.UUID(cursorID),
// 		OlderLimit:      halfLimit + 1,
// 		NewerLimit:      halfLimit,
// 	})
// 	if err != nil {
// 		return nil, db.NewError(err, db.EntityMessage)
// 	}

// 	return messagesFromRows(rows)
// }

// func (r *Channel) MessageListByChannelBefore(
// 	ctx context.Context,
// 	channelID uuid.UUID,
// 	cursorCreatedAt *time.Time,
// 	cursorID *uuid.UUID,
// 	limit int32,
// ) ([]channel.Message, error) {
// 	rows, err := r.store.MessageListByChannelBefore(ctx, db.MessageListByChannelBeforeParams{
// 		ChannelID:       db.UUID(channelID),
// 		CursorCreatedAt: db.TimestamptzPtr(cursorCreatedAt),
// 		CursorID:        db.UUIDPtr(cursorID),
// 		ResultLimit:     limit,
// 	})
// 	if err != nil {
// 		return nil, db.NewError(err, db.EntityMessage)
// 	}

// 	return messagesFromRows(rows)
// }

// func (r *Channel) MessageListPinnedByChannel(ctx context.Context, channelID uuid.UUID) ([]*channel.Message, error) {
// 	rows, err := r.store.MessageListPinnedByChannel(ctx, db.UUID(channelID))
// 	if err != nil {
// 		return nil, db.NewError(err, db.EntityMessage)
// 	}

// 	return messagesFromRows(rows)
// }

func (r *Channel) MessageSetPinned(ctx context.Context, msg *channel.Message) (*channel.Message, error) {
	row, err := r.store.MessageSetPinned(ctx, db.MessageSetPinnedParams{
		ID:        db.UUID(msg.ID().UUID()),
		IsPinned:  msg.IsPinned(),
		UpdatedAt: db.Timestamptz(msg.UpdatedAt()),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	return messageFromRow(row)
}

func (r *Channel) MessageUpdateContent(ctx context.Context, msg *channel.Message) (*channel.Message, error) {
	row, err := r.store.MessageUpdateContent(ctx, db.MessageUpdateContentParams{
		ID:       db.UUID(uuid.UUID(msg.ID())),
		Content:  db.Text(msg.Content().String()),
		EditedAt: db.Timestamptz(*msg.EditedAt()),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	return messageFromRow(row)
}

// ============================================================================
// REACTIONS
// ============================================================================

func (r *Channel) ReactionAdd(ctx context.Context, messageID, userID uuid.UUID, emoji string) (*channel.Reaction, error) {
	row, err := r.store.ReactionAdd(ctx, db.ReactionAddParams{
		MessageID: db.UUID(messageID),
		UserID:    db.UUID(userID),
		Emoji:     emoji,
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityMessageReaction)
	}

	return reactionFromRow(row)
}

func (r *Channel) ReactionRemove(ctx context.Context, messageID, userID uuid.UUID, emoji channel.Emoji) error {
	err := r.store.ReactionRemove(ctx, db.ReactionRemoveParams{
		MessageID: db.UUID(messageID),
		UserID:    db.UUID(userID),
		Emoji:     emoji.String(),
	})
	if err != nil {
		return db.NewError(err, db.EntityMessageReaction)
	}

	return nil
}

// ============================================================================
// INTERNAL HELPERS
// ============================================================================

func channelFromRow(row db.Channel) (*channel.Channel, error) {
	ch, err := channel.Reconstitute(
		row.ID.Bytes,
		row.Type,
		db.StringPtr(row.Name),
		db.StringPtr(row.IconUrl),
		db.UUIDPtrFromDB(row.LastMessageID),
		row.CreatedAt.Time.UTC(),
		row.UpdatedAt.Time.UTC(),
	)
	if err != nil {
		return nil, db.NewError(err, db.EntityChannel)
	}

	return ch, nil
}

func memberFromRow(row db.ChannelMember) (*channel.Member, error) {
	mem, err := channel.ReconstituteMember(
		row.ChannelID.Bytes,
		row.UserID.Bytes,
		db.UUIDPtrFromDB(row.LastReadMessageID),
		row.MentionCount,
		row.LastReadAt.Time.UTC(),
		row.CreatedAt.Time.UTC(),
		row.UpdatedAt.Time.UTC(),
	)
	if err != nil {
		return nil, db.NewError(err, db.EntityChannelMember)
	}

	return mem, nil
}

func memberListItemFromRow(row db.ChannelMemberListItemsByChannelRow) (*channel.MemberListItem, error) {
	mem, err := channel.ReconstituteMemberListItem(
		row.ChannelID.Bytes,
		row.UserID.Bytes,
		row.MemberSince.Time.UTC(),
		row.LastReadAt.Time.UTC(),
		row.Username,
		row.DisplayName,
		db.StringPtr(row.AvatarUrl),
		row.UserCreatedAt.Time.UTC(),
	)
	if err != nil {
		return nil, db.NewError(err, db.EntityChannelMember)
	}

	return mem, nil
}

func messageFromRow(row db.Message) (*channel.Message, error) {
	msg, err := channel.ReconstituteMessage(
		row.ID.Bytes,
		row.ChannelID.Bytes,
		db.UUIDPtrFromDB(row.AuthorID),
		db.UUIDPtrFromDB(row.ReplyToMessageID),
		db.StringPtr(row.Content),
		row.IsPinned,
		row.CreatedAt.Time.UTC(),
		row.UpdatedAt.Time.UTC(),
		db.TimePtr(row.EditedAt),
	)
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	return msg, nil
}

func messageAggregateFromRow(row db.MessageGetAggregateRow, currentUserID *uuid.UUID) (*channel.MessageAggregate, error) {
	// 1. Reconstitute Base Message
	msg, err := channel.ReconstituteMessage(
		row.ID.Bytes,
		row.ChannelID.Bytes,
		db.UUIDPtrFromDB(row.AuthorID),
		db.UUIDPtrFromDB(row.ReplyToMessageID),
		db.StringPtr(row.Content),
		row.IsPinned,
		row.CreatedAt.Time.UTC(),
		row.UpdatedAt.Time.UTC(),
		db.TimePtr(row.EditedAt),
	)
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	// 2. Reconstitute Author Summary
	authorID := db.UUIDPtrFromDB(row.AuthorID)
	var author channel.AuthorSummary
	if authorID != nil {
		var err error
		author, err = channel.ReconstituteAuthorSummary(
			authorID,
			row.AuthorUsername.String,
			row.AuthorDisplayName.String,
			db.StringPtr(row.AuthorAvatarUrl),
		)
		if err != nil {
			return nil, db.NewError(err, db.EntityMessage)
		}
	}

	// 3. Unmarshal Attachments from JSON
	attachments, err := channel.UnmarshalAttachmentsJSON(row.ID.Bytes, row.Attachments)
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	// 4. Unmarshal and aggregate ReactionSummaries from JSON
	reactions, err := channel.UnmarshalReactionsJSON(row.Reactions, currentUserID)
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	// 5. Build Final Domain Aggregate
	agg, err := channel.ReconstituteMessageAggregate(msg, author, attachments, reactions)
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	return agg, nil
}

func messagesFromRows(rows []db.Message) ([]*channel.Message, error) {
	messages := make([]*channel.Message, 0, len(rows))
	for _, row := range rows {
		msg, err := messageFromRow(row)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

func reactionFromRow(row db.MessageReaction) (*channel.Reaction, error) {
	return channel.ReconstituteReaction(
		row.MessageID.Bytes,
		row.UserID.Bytes,
		row.Emoji,
		row.CreatedAt.Time.UTC(),
	)
}
