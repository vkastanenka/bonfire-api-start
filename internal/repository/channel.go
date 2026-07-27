package repository

import (
	"context"
	"errors"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ChannelStore interface {
	// Channels
	ChannelCreate(ctx context.Context, arg db.ChannelCreateParams) (db.Channel, error)
	ChannelGet(ctx context.Context, id pgtype.UUID) (db.Channel, error)
	ChannelGetForUpdate(ctx context.Context, id pgtype.UUID) (db.Channel, error)
	ChannelUpdate(ctx context.Context, arg db.ChannelUpdateParams) (db.Channel, error)
	ChannelUpdateLastMessage(ctx context.Context, arg db.ChannelUpdateLastMessageParams) error
	ChannelDelete(ctx context.Context, id pgtype.UUID) error
	ChannelFindDirectMessage(ctx context.Context, arg db.ChannelFindDirectMessageParams) (db.Channel, error)

	// Channel Members
	ChannelMemberAdd(ctx context.Context, arg db.ChannelMemberAddParams) (db.ChannelMember, error)
	ChannelMemberGet(ctx context.Context, arg db.ChannelMemberGetParams) (db.ChannelMember, error)
	ChannelMemberListByChannel(ctx context.Context, channelID pgtype.UUID) ([]db.ChannelMember, error)
	ChannelMemberListByUser(ctx context.Context, userID pgtype.UUID) ([]db.ChannelMemberListByUserRow, error)
	ChannelMemberUpdateReadState(ctx context.Context, arg db.ChannelMemberUpdateReadStateParams) error
	ChannelMemberIncrementMentionCount(ctx context.Context, arg db.ChannelMemberIncrementMentionCountParams) error
	ChannelMemberRemove(ctx context.Context, arg db.ChannelMemberRemoveParams) error

	// Messages
	MessageCreate(ctx context.Context, arg db.MessageCreateParams) (db.Message, error)
	MessageGet(ctx context.Context, id pgtype.UUID) (db.Message, error)
	MessageListByChannelKeyset(ctx context.Context, arg db.MessageListByChannelKeysetParams) ([]db.Message, error)
	MessageListPinnedByChannel(ctx context.Context, channelID pgtype.UUID) ([]db.Message, error)
	MessageUpdateContent(ctx context.Context, arg db.MessageUpdateContentParams) (db.Message, error)
	MessageSetPinned(ctx context.Context, arg db.MessageSetPinnedParams) error
	MessageDelete(ctx context.Context, id pgtype.UUID) error

	// Reactions
	ReactionAdd(ctx context.Context, arg db.ReactionAddParams) (db.MessageReaction, error)
	ReactionRemove(ctx context.Context, arg db.ReactionRemoveParams) error
	ReactionListByMessage(ctx context.Context, messageID pgtype.UUID) ([]db.MessageReaction, error)
	ReactionSummarizeByMessage(ctx context.Context, messageID pgtype.UUID) ([]db.ReactionSummarizeByMessageRow, error)
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

func (r *Channel) CreateChannel(ctx context.Context, ch *channel.Channel) error {
	_, err := r.store.ChannelCreate(ctx, db.ChannelCreateParams{
		ID:        db.UUID(ch.ID()),
		Type:      int16(ch.Type()),
		OwnerID:   db.UUIDPtr(ch.OwnerID()),
		Name:      db.TextPtr(ch.Name()),
		IconUrl:   db.TextPtr(ch.IconURL()),
		CreatedAt: db.Timestamptz(ch.CreatedAt()),
		UpdatedAt: db.Timestamptz(ch.UpdatedAt()),
	})
	if err != nil {
		return db.NewError(err, db.EntityChannel)
	}

	return nil
}

func (r *Channel) GetChannel(ctx context.Context, id uuid.UUID) (*channel.Channel, error) {
	row, err := r.store.ChannelGet(ctx, db.UUID(id))
	if err != nil {
		return nil, db.NewError(err, db.EntityChannel)
	}

	return channelFromRow(row), nil
}

func (r *Channel) UpdateChannel(ctx context.Context, ch *channel.Channel) error {
	_, err := r.store.ChannelUpdate(ctx, db.ChannelUpdateParams{
		ID:            db.UUID(ch.ID()),
		Name:          db.TextPtr(ch.Name()),
		IconUrl:       db.TextPtr(ch.IconURL()),
		OwnerID:       db.UUIDPtr(ch.OwnerID()),
		LastMessageID: db.UUIDPtr(ch.LastMessageID()),
		UpdatedAt:     db.Timestamptz(ch.UpdatedAt()),
	})
	if err != nil {
		return db.NewError(err, db.EntityChannel)
	}

	return nil
}

func (r *Channel) FindDirectMessage(ctx context.Context, user1ID, user2ID uuid.UUID) (*channel.Channel, error) {
	row, err := r.store.ChannelFindDirectMessage(ctx, db.ChannelFindDirectMessageParams{
		User1ID: db.UUID(user1ID),
		User2ID: db.UUID(user2ID),
	})
	if err != nil {
		if db.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, db.NewError(err, db.EntityChannel)
	}

	return channelFromRow(row), nil
}

func (r *Channel) IsParticipant(ctx context.Context, channelID, userID uuid.UUID) (bool, error) {
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

	return memberFromRow(row), nil
}

func (r *Channel) UpdateMemberReadState(ctx context.Context, channelID, userID, messageID uuid.UUID) error {
	err := r.store.ChannelMemberUpdateReadState(ctx, db.ChannelMemberUpdateReadStateParams{
		ChannelID:         db.UUID(channelID),
		UserID:            db.UUID(userID),
		LastReadMessageID: db.UUIDPtr(&messageID),
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
		Content:          db.Text(msg.Content()),
		IsPinned:         msg.IsPinned(),
		CreatedAt:        db.Timestamptz(msg.CreatedAt()),
		EditedAt:         db.TimestamptzPtr(msg.EditedAt()),
	})
	if err != nil {
		return db.NewError(err, db.EntityMessage)
	}

	// Update channel last message reference
	err = r.store.ChannelUpdateLastMessage(ctx, db.ChannelUpdateLastMessageParams{
		ChannelID:     db.UUID(msg.ChannelID()),
		LastMessageID: db.UUID(msg.ID()),
		UpdatedAt:     db.Timestamptz(msg.CreatedAt()),
	})
	if err != nil {
		return db.NewError(err, db.EntityChannel)
	}

	return nil
}

func (r *Channel) GetMessage(ctx context.Context, id uuid.UUID) (*channel.Message, error) {
	row, err := r.store.MessageGet(ctx, db.UUID(id))
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	return messageFromRow(row), nil
}

func (r *Channel) UpdateMessage(ctx context.Context, msg *channel.Message) error {
	if msg.EditedAt() == nil {
		return errors.New("editedAt timestamp is required to update message")
	}

	_, err := r.store.MessageUpdateContent(ctx, db.MessageUpdateContentParams{
		ID:       db.UUID(msg.ID()),
		Content:  db.Text(msg.Content()),
		EditedAt: db.Timestamptz(*msg.EditedAt()),
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

func (r *Channel) ListMessages(ctx context.Context, channelID uuid.UUID, before *uuid.UUID, limit int) ([]channel.Message, error) {
	rows, err := r.store.MessageListByChannelKeyset(ctx, db.MessageListByChannelKeysetParams{
		ChannelID:   db.UUID(channelID),
		CursorID:    db.UUIDPtr(before),
		ResultLimit: int32(limit),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	messages := make([]channel.Message, len(rows))
	for i, row := range rows {
		messages[i] = *messageFromRow(row)
	}
	return messages, nil
}

func (r *Channel) ListPinnedMessages(ctx context.Context, channelID uuid.UUID) ([]channel.Message, error) {
	rows, err := r.store.MessageListPinnedByChannel(ctx, db.UUID(channelID))
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	messages := make([]channel.Message, len(rows))
	for i, row := range rows {
		messages[i] = *messageFromRow(row)
	}
	return messages, nil
}

// ============================================================================
// REACTIONS
// ============================================================================

func (r *Channel) AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	emojiVO, err := channel.NewEmoji(emoji)
	if err != nil {
		return err
	}

	reaction, err := channel.NewReaction(messageID, userID, emojiVO)
	if err != nil {
		return err
	}

	_, err = r.store.ReactionAdd(ctx, db.ReactionAddParams{
		MessageID: db.UUID(reaction.MessageID()),
		UserID:    db.UUID(reaction.UserID()),
		Emoji:     reaction.Emoji(),
		CreatedAt: db.Timestamptz(reaction.CreatedAt()),
	})
	if err != nil {
		return db.NewError(err, db.EntityMessageReaction)
	}

	return nil
}

func (r *Channel) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	err := r.store.ReactionRemove(ctx, db.ReactionRemoveParams{
		MessageID: db.UUID(messageID),
		UserID:    db.UUID(userID),
		Emoji:     emoji,
	})
	if err != nil {
		return db.NewError(err, db.EntityMessageReaction)
	}
	return nil
}

func (r *Channel) GetReactionSummaries(ctx context.Context, messageID uuid.UUID) ([]channel.ReactionSummary, error) {
	rows, err := r.store.ReactionSummarizeByMessage(ctx, db.UUID(messageID))
	if err != nil {
		return nil, db.NewError(err, db.EntityMessageReaction)
	}

	summaries := make([]channel.ReactionSummary, len(rows))
	for i, row := range rows {
		summaries[i] = channel.ReconstituteReactionSummary(row.Emoji, int64(row.Count))
	}
	return summaries, nil
}

// ============================================================================
// INTERNAL HELPERS
// ============================================================================

func channelFromRow(row db.Channel) *channel.Channel {
	return channel.Reconstitute(
		uuid.UUID(row.ID.Bytes),
		channel.Type(row.Type),
		db.UUIDPtrFromDB(row.OwnerID),
		db.StringPtr(row.Name),
		db.StringPtr(row.IconUrl),
		db.UUIDPtrFromDB(row.LastMessageID),
		row.CreatedAt.Time.UTC(),
		row.UpdatedAt.Time.UTC(),
	)
}

func memberFromRow(row db.ChannelMember) *channel.Member {
	return channel.ReconstituteMember(
		uuid.UUID(row.ChannelID.Bytes),
		uuid.UUID(row.UserID.Bytes),
		row.JoinedAt.Time.UTC(),
		db.UUIDPtrFromDB(row.LastReadMessageID),
		row.MentionCount,
	)
}

func messageFromRow(row db.Message) *channel.Message {
	return channel.ReconstituteMessage(
		uuid.UUID(row.ID.Bytes),
		uuid.UUID(row.ChannelID.Bytes),
		db.UUIDPtrFromDB(row.AuthorID),
		db.UUIDPtrFromDB(row.ReplyToMessageID),
		row.Content.String,
		row.IsPinned,
		row.CreatedAt.Time.UTC(),
		db.TimePtr(row.EditedAt),
	)
}
