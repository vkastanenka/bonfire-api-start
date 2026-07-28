package repository

import (
	"context"
	"errors"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ChannelStore interface {
	// ============================================================================
	// CHANNELS
	// ============================================================================
	ChannelCreate(ctx context.Context, arg db.ChannelCreateParams) (db.Channel, error)
	ChannelGet(ctx context.Context, id pgtype.UUID) (db.Channel, error)
	ChannelUpdate(ctx context.Context, arg db.ChannelUpdateParams) (db.Channel, error)
	ChannelUpdateLastMessage(ctx context.Context, arg db.ChannelUpdateLastMessageParams) error
	ChannelDelete(ctx context.Context, id pgtype.UUID) error
	ChannelFindDM(ctx context.Context, arg db.ChannelFindDMParams) (db.Channel, error)
	ChannelListByUser(ctx context.Context, userID pgtype.UUID) ([]db.ChannelListByUserRow, error)

	// ============================================================================
	// CHANNEL MEMBERS
	// ============================================================================
	ChannelMemberAdd(ctx context.Context, arg db.ChannelMemberAddParams) (db.ChannelMember, error)
	ChannelMemberGet(ctx context.Context, arg db.ChannelMemberGetParams) (db.ChannelMember, error)
	ChannelMemberListByChannel(ctx context.Context, channelID pgtype.UUID) ([]db.ChannelMemberListByChannelRow, error)
	ChannelMemberUpdateReadState(ctx context.Context, arg db.ChannelMemberUpdateReadStateParams) error
	ChannelMemberIncrementMentionCount(ctx context.Context, arg db.ChannelMemberIncrementMentionCountParams) error
	ChannelMemberIncrementMentionCountBatch(ctx context.Context, arg db.ChannelMemberIncrementMentionCountBatchParams) error
	ChannelMemberRemove(ctx context.Context, arg db.ChannelMemberRemoveParams) error

	// ============================================================================
	// MESSAGES
	// ============================================================================
	MessageCreate(ctx context.Context, arg db.MessageCreateParams) (db.Message, error)
	MessageGet(ctx context.Context, id pgtype.UUID) (db.Message, error)
	MessageListByChannelKeyset(ctx context.Context, arg db.MessageListByChannelKeysetParams) ([]db.Message, error)
	MessageListPinnedByChannel(ctx context.Context, channelID pgtype.UUID) ([]db.Message, error)
	MessageListReplies(ctx context.Context, replyToMessageID pgtype.UUID) ([]db.Message, error)
	MessageUpdateContent(ctx context.Context, arg db.MessageUpdateContentParams) (db.Message, error)
	MessageSetPinned(ctx context.Context, arg db.MessageSetPinnedParams) error
	MessageDelete(ctx context.Context, id pgtype.UUID) error

	// ============================================================================
	// MESSAGE REACTIONS
	// ============================================================================
	ReactionAdd(ctx context.Context, arg db.ReactionAddParams) (db.MessageReaction, error)
	ReactionRemove(ctx context.Context, arg db.ReactionRemoveParams) error
	ReactionListByMessage(ctx context.Context, messageID pgtype.UUID) ([]db.MessageReaction, error)
	ReactionSummarizeByMessage(ctx context.Context, arg db.ReactionSummarizeByMessageParams) ([]db.ReactionSummarizeByMessageRow, error)
}

type Channel struct {
	store ChannelStore
}

func NewChannel(store ChannelStore) *Channel {
	return &Channel{store: store}
}

// ============================================================================
// CHANNELS
// ============================================================================

func (r *Channel) Create(ctx context.Context, ch *channel.Channel) error {
	_, err := r.store.ChannelCreate(ctx, db.ChannelCreateParams{
		ID:        db.UUID(ch.ID()),
		Type:      int16(ch.Type()),
		OwnerID:   db.UUIDPtr(ch.OwnerID()),
		Name:      db.StringerPtr(ch.Name()),
		IconUrl:   db.StringerPtr(ch.IconURL()),
		CreatedAt: db.Timestamptz(ch.CreatedAt()),
		UpdatedAt: db.Timestamptz(ch.UpdatedAt()),
	})
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

func (r *Channel) Update(ctx context.Context, ch *channel.Channel) error {
	_, err := r.store.ChannelUpdate(ctx, db.ChannelUpdateParams{
		ID:            db.UUID(ch.ID()),
		Name:          db.StringerPtr(ch.Name()),
		IconUrl:       db.StringerPtr(ch.IconURL()),
		OwnerID:       db.UUIDPtr(ch.OwnerID()),
		LastMessageID: db.UUIDPtr(ch.LastMessageID()),
		UpdatedAt:     db.Timestamptz(ch.UpdatedAt()),
	})
	if err != nil {
		return db.NewError(err, db.EntityChannel)
	}

	return nil
}

func (r *Channel) UpdateLastMessage(ctx context.Context, channelID, messageID uuid.UUID, updatedAt time.Time) error {
	err := r.store.ChannelUpdateLastMessage(ctx, db.ChannelUpdateLastMessageParams{
		ChannelID:     db.UUID(channelID),
		LastMessageID: db.UUID(messageID),
		UpdatedAt:     db.Timestamptz(updatedAt),
	})
	if err != nil {
		return db.NewError(err, db.EntityChannel)
	}

	return nil
}

func (r *Channel) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.ChannelDelete(ctx, db.UUID(id))
	if err != nil {
		return db.NewError(err, db.EntityChannel)
	}

	return nil
}

func (r *Channel) FindDM(ctx context.Context, user1ID, user2ID uuid.UUID) (*channel.Channel, error) {
	u1, u2 := user1ID, user2ID
	if u1.String() > u2.String() {
		u1, u2 = u2, u1
	}

	row, err := r.store.ChannelFindDM(ctx, db.ChannelFindDMParams{
		User1ID: db.UUID(u1),
		User2ID: db.UUID(u2),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityChannel)
	}

	return channelFromRow(row)
}

func (r *Channel) ListByUser(ctx context.Context, userID uuid.UUID) ([]db.ChannelListByUserRow, error) {
	rows, err := r.store.ChannelListByUser(ctx, db.UUID(userID))
	if err != nil {
		return nil, db.NewError(err, db.EntityChannel)
	}

	return rows, nil
}

// ============================================================================
// MEMBERS
// ============================================================================

func (r *Channel) AddMember(ctx context.Context, m *channel.Member) error {
	_, err := r.store.ChannelMemberAdd(ctx, db.ChannelMemberAddParams{
		ChannelID:         db.UUID(m.ChannelID()),
		UserID:            db.UUID(m.UserID()),
		JoinedAt:          db.Timestamptz(m.JoinedAt()),
		LastReadMessageID: db.UUIDPtr(m.LastReadMessageID()),
		MentionCount:      m.MentionCount(),
	})
	if err != nil {
		return db.NewError(err, db.EntityChannelMember)
	}

	return nil
}

func (r *Channel) GetMember(ctx context.Context, channelID, userID uuid.UUID) (*channel.Member, error) {
	row, err := r.store.ChannelMemberGet(ctx, db.ChannelMemberGetParams{
		ChannelID: db.UUID(channelID),
		UserID:    db.UUID(userID),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityChannelMember)
	}

	return memberFromRow(row)
}

func (r *Channel) ListMembers(ctx context.Context, channelID uuid.UUID) ([]db.ChannelMemberListByChannelRow, error) {
	rows, err := r.store.ChannelMemberListByChannel(ctx, db.UUID(channelID))
	if err != nil {
		return nil, db.NewError(err, db.EntityChannelMember)
	}

	return rows, nil
}

func (r *Channel) UpdateMemberReadState(ctx context.Context, channelID, userID, messageID uuid.UUID) error {
	err := r.store.ChannelMemberUpdateReadState(ctx, db.ChannelMemberUpdateReadStateParams{
		ChannelID:         db.UUID(channelID),
		UserID:            db.UUID(userID),
		LastReadMessageID: db.UUID(messageID),
	})
	if err != nil {
		return db.NewError(err, db.EntityChannelMember)
	}

	return nil
}

func (r *Channel) IncrementMentionCount(ctx context.Context, channelID, userID uuid.UUID) error {
	err := r.store.ChannelMemberIncrementMentionCount(ctx, db.ChannelMemberIncrementMentionCountParams{
		ChannelID: db.UUID(channelID),
		UserID:    db.UUID(userID),
	})
	if err != nil {
		return db.NewError(err, db.EntityChannelMember)
	}

	return nil
}

func (r *Channel) IncrementMentionCountBatch(ctx context.Context, channelID uuid.UUID, userIDs []uuid.UUID) error {
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

func (r *Channel) RemoveMember(ctx context.Context, channelID, userID uuid.UUID) error {
	err := r.store.ChannelMemberRemove(ctx, db.ChannelMemberRemoveParams{
		ChannelID: db.UUID(channelID),
		UserID:    db.UUID(userID),
	})
	if err != nil {
		return db.NewError(err, db.EntityChannelMember)
	}

	return nil
}

// ============================================================================
// MESSAGES
// ============================================================================

func (r *Channel) CreateMessage(ctx context.Context, msg *channel.Message) error {
	_, err := r.store.MessageCreate(ctx, db.MessageCreateParams{
		ID:               db.UUID(msg.ID()),
		ChannelID:        db.UUID(msg.ChannelID()),
		AuthorID:         db.UUIDPtr(msg.AuthorID()),
		ReplyToMessageID: db.UUIDPtr(msg.ReplyToMessageID()),
		Content:          db.Text(msg.Content().String()),
		IsPinned:         msg.IsPinned(),
		CreatedAt:        db.Timestamptz(msg.CreatedAt()),
	})
	if err != nil {
		return db.NewError(err, db.EntityMessage)
	}

	return nil
}

func (r *Channel) GetMessage(ctx context.Context, id uuid.UUID) (*channel.Message, error) {
	row, err := r.store.MessageGet(ctx, db.UUID(id))
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	return messageFromRow(row)
}

func (r *Channel) ListMessages(
	ctx context.Context,
	channelID uuid.UUID,
	cursorCreatedAt *time.Time,
	cursorID *uuid.UUID,
	limit int,
) ([]channel.Message, error) {
	rows, err := r.store.MessageListByChannelKeyset(ctx, db.MessageListByChannelKeysetParams{
		ChannelID:       db.UUID(channelID),
		CursorCreatedAt: db.TimestamptzPtr(cursorCreatedAt),
		CursorID:        db.UUIDPtr(cursorID),
		ResultLimit:     int32(limit),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	messages := make([]channel.Message, 0, len(rows))
	for _, row := range rows {
		msg, err := messageFromRow(row)
		if err != nil {
			return nil, db.NewError(err, db.EntityMessage)
		}
		messages = append(messages, *msg)
	}

	return messages, nil
}

func (r *Channel) ListPinnedMessages(ctx context.Context, channelID uuid.UUID) ([]channel.Message, error) {
	rows, err := r.store.MessageListPinnedByChannel(ctx, db.UUID(channelID))
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	messages := make([]channel.Message, 0, len(rows))
	for _, row := range rows {
		msg, err := messageFromRow(row)
		if err != nil {
			return nil, db.NewError(err, db.EntityMessage)
		}
		messages = append(messages, *msg)
	}

	return messages, nil
}

func (r *Channel) ListMessageReplies(ctx context.Context, replyToMessageID uuid.UUID) ([]channel.Message, error) {
	rows, err := r.store.MessageListReplies(ctx, db.UUID(replyToMessageID))
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	messages := make([]channel.Message, 0, len(rows))
	for _, row := range rows {
		msg, err := messageFromRow(row)
		if err != nil {
			return nil, db.NewError(err, db.EntityMessage)
		}
		messages = append(messages, *msg)
	}

	return messages, nil
}

func (r *Channel) UpdateMessage(ctx context.Context, msg *channel.Message) error {
	if msg.EditedAt() == nil {
		return errors.New("editedAt timestamp is required to update message")
	}

	_, err := r.store.MessageUpdateContent(ctx, db.MessageUpdateContentParams{
		ID:       db.UUID(msg.ID()),
		Content:  db.Text(msg.Content().String()),
		EditedAt: db.Timestamptz(*msg.EditedAt()),
	})
	if err != nil {
		return db.NewError(err, db.EntityMessage)
	}

	return nil
}

func (r *Channel) SetPinnedMessage(ctx context.Context, id uuid.UUID, isPinned bool) error {
	err := r.store.MessageSetPinned(ctx, db.MessageSetPinnedParams{
		ID:       db.UUID(id),
		IsPinned: isPinned,
	})
	if err != nil {
		return db.NewError(err, db.EntityMessage)
	}

	return nil
}

func (r *Channel) DeleteMessage(ctx context.Context, id uuid.UUID) error {
	err := r.store.MessageDelete(ctx, db.UUID(id))
	if err != nil {
		return db.NewError(err, db.EntityMessage)
	}

	return nil
}

// ============================================================================
// REACTIONS
// ============================================================================

func (r *Channel) AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) (*channel.Reaction, error) {
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

func (r *Channel) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji channel.Emoji) error {
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

func (r *Channel) ListReactionsByMessage(ctx context.Context, messageID uuid.UUID) ([]channel.Reaction, error) {
	rows, err := r.store.ReactionListByMessage(ctx, db.UUID(messageID))
	if err != nil {
		return nil, db.NewError(err, db.EntityMessageReaction)
	}

	reactions := make([]channel.Reaction, 0, len(rows))
	for _, row := range rows {
		react, err := reactionFromRow(row)
		if err != nil {
			return nil, db.NewError(err, db.EntityMessageReaction)
		}
		reactions = append(reactions, *react)
	}

	return reactions, nil
}

func (r *Channel) SummarizeReactionsByMessage(ctx context.Context, messageID, currentUserID uuid.UUID) ([]channel.ReactionSummary, error) {
	rows, err := r.store.ReactionSummarizeByMessage(ctx, db.ReactionSummarizeByMessageParams{
		MessageID:     db.UUID(messageID),
		CurrentUserID: db.UUID(currentUserID),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityMessageReaction)
	}

	summaries := make([]channel.ReactionSummary, 0, len(rows))
	for _, row := range rows {
		summary, err := channel.ReconstituteReactionSummary(
			row.Emoji,
			int64(row.Count),
			row.MeReacted,
		)
		if err != nil {
			return nil, db.NewError(err, db.EntityMessageReaction)
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// ============================================================================
// INTERNAL HELPERS
// ============================================================================

func channelFromRow(row db.Channel) (*channel.Channel, error) {
	return channel.Reconstitute(
		row.ID.Bytes,
		channel.Type(row.Type),
		db.UUIDPtrFromDB(row.OwnerID),
		db.StringPtr(row.Name),
		db.StringPtr(row.IconUrl),
		db.UUIDPtrFromDB(row.LastMessageID),
		row.CreatedAt.Time.UTC(),
		row.UpdatedAt.Time.UTC(),
	)
}

func memberFromRow(row db.ChannelMember) (*channel.Member, error) {
	mem, err := channel.ReconstituteMember(
		row.ChannelID.Bytes,
		row.UserID.Bytes,
		row.JoinedAt.Time.UTC(),
		db.UUIDPtrFromDB(row.LastReadMessageID),
		row.MentionCount,
	)
	if err != nil {
		return nil, db.NewError(err, db.EntityChannelMember)
	}

	return mem, nil
}

func messageFromRow(row db.Message) (*channel.Message, error) {
	return channel.ReconstituteMessage(
		row.ID.Bytes,
		row.ChannelID.Bytes,
		db.UUIDPtrFromDB(row.AuthorID),
		db.UUIDPtrFromDB(row.ReplyToMessageID),
		row.Content.String,
		row.IsPinned,
		row.CreatedAt.Time.UTC(),
		db.TimePtr(row.EditedAt),
	)
}

func reactionFromRow(row db.MessageReaction) (*channel.Reaction, error) {
	return channel.ReconstituteReaction(
		row.MessageID.Bytes,
		row.UserID.Bytes,
		row.Emoji,
		row.CreatedAt.Time.UTC(),
	)
}

// func (r *Channel) IsParticipant(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
// 	_, err := r.store.ChannelMemberGet(ctx, db.ChannelMemberGetParams{
// 		ChannelID: db.UUID(channelID),
// 		UserID:    db.UUID(userID),
// 	})
// 	if err != nil {
// 		if db.IsNotFoundError(err) {
// 			return false, nil
// 		}
// 		return false, db.NewError(err, db.EntityChannelMember)
// 	}
// 	return true, nil
// }
