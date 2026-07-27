package repository

import (
	"context"
	"time"

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

func (r *Channel) Create(ctx context.Context, ch *channel.Channel) error {
	row, err := r.store.ChannelCreate(ctx, db.ChannelCreateParams{
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

	*ch = *channelFromRow(row)
	return nil
}

func (r *Channel) Get(ctx context.Context, id uuid.UUID) (*channel.Channel, error) {
	row, err := r.store.ChannelGet(ctx, db.UUID(id))
	if err != nil {
		return nil, db.NewError(err, db.EntityChannel)
	}

	return channelFromRow(row), nil
}

func (r *Channel) GetForUpdate(ctx context.Context, id uuid.UUID) (*channel.Channel, error) {
	row, err := r.store.ChannelGetForUpdate(ctx, db.UUID(id))
	if err != nil {
		return nil, db.NewError(err, db.EntityChannel)
	}

	return channelFromRow(row), nil
}

func (r *Channel) Update(ctx context.Context, ch *channel.Channel) error {
	row, err := r.store.ChannelUpdate(ctx, db.ChannelUpdateParams{
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

	*ch = *channelFromRow(row)
	return nil
}

func (r *Channel) UpdateLastMessage(ctx context.Context, channelID uuid.UUID, lastMessageID *uuid.UUID) error {
	err := r.store.ChannelUpdateLastMessage(ctx, db.ChannelUpdateLastMessageParams{
		ChannelID:     db.UUID(channelID),
		LastMessageID: db.UUIDPtr(lastMessageID),
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

func (r *Channel) FindDirectMessage(ctx context.Context, user1ID, user2ID uuid.UUID) (*channel.Channel, error) {
	row, err := r.store.ChannelFindDirectMessage(ctx, db.ChannelFindDirectMessageParams{
		User1ID: db.UUID(user1ID),
		User2ID: db.UUID(user2ID),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityChannel)
	}

	return channelFromRow(row), nil
}

// ============================================================================
// CHANNEL MEMBERS
// ============================================================================

func (r *Channel) MemberAdd(ctx context.Context, m *channel.Member) error {
	row, err := r.store.ChannelMemberAdd(ctx, db.ChannelMemberAddParams{
		ChannelID:         db.UUID(m.ChannelID()),
		UserID:            db.UUID(m.UserID()),
		JoinedAt:          db.Timestamptz(m.JoinedAt()),
		LastReadMessageID: db.UUIDPtr(m.LastReadMessageID()),
		MentionCount:      m.MentionCount(),
	})
	if err != nil {
		return db.NewError(err, db.EntityChannelMember)
	}

	*m = *memberFromRow(row)
	return nil
}

func (r *Channel) MemberGet(ctx context.Context, channelID, userID uuid.UUID) (*channel.Member, error) {
	row, err := r.store.ChannelMemberGet(ctx, db.ChannelMemberGetParams{
		ChannelID: db.UUID(channelID),
		UserID:    db.UUID(userID),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityChannelMember)
	}

	return memberFromRow(row), nil
}

func (r *Channel) MemberListByChannel(ctx context.Context, channelID uuid.UUID) ([]*channel.Member, error) {
	rows, err := r.store.ChannelMemberListByChannel(ctx, db.UUID(channelID))
	if err != nil {
		return nil, db.NewError(err, db.EntityChannelMember)
	}

	members := make([]*channel.Member, len(rows))
	for i, row := range rows {
		members[i] = memberFromRow(row)
	}
	return members, nil
}

func (r *Channel) MemberListByUser(ctx context.Context, userID uuid.UUID) ([]*channel.UserSidebarItem, error) {
	rows, err := r.store.ChannelMemberListByUser(ctx, db.UUID(userID))
	if err != nil {
		return nil, db.NewError(err, db.EntityChannelMember)
	}

	items := make([]*channel.UserSidebarItem, len(rows))
	for i, row := range rows {
		items[i] = sidebarItemFromRow(row)
	}
	return items, nil
}

func (r *Channel) MemberUpdateReadState(ctx context.Context, channelID, userID uuid.UUID, lastReadMessageID *uuid.UUID) error {
	err := r.store.ChannelMemberUpdateReadState(ctx, db.ChannelMemberUpdateReadStateParams{
		ChannelID:         db.UUID(channelID),
		UserID:            db.UUID(userID),
		LastReadMessageID: db.UUIDPtr(lastReadMessageID),
	})
	if err != nil {
		return db.NewError(err, db.EntityChannelMember)
	}
	return nil
}

func (r *Channel) MemberIncrementMentionCount(ctx context.Context, channelID, userID uuid.UUID) error {
	err := r.store.ChannelMemberIncrementMentionCount(ctx, db.ChannelMemberIncrementMentionCountParams{
		ChannelID: db.UUID(channelID),
		UserID:    db.UUID(userID),
	})
	if err != nil {
		return db.NewError(err, db.EntityChannelMember)
	}
	return nil
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

// ============================================================================
// MESSAGES
// ============================================================================

func (r *Channel) MessageCreate(ctx context.Context, msg *channel.Message) error {
	row, err := r.store.MessageCreate(ctx, db.MessageCreateParams{
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

	*msg = *messageFromRow(row)
	return nil
}

func (r *Channel) MessageGet(ctx context.Context, id uuid.UUID) (*channel.Message, error) {
	row, err := r.store.MessageGet(ctx, db.UUID(id))
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	return messageFromRow(row), nil
}

func (r *Channel) MessageListByChannelKeyset(ctx context.Context, channelID uuid.UUID, cursorID *uuid.UUID, limit int32) ([]*channel.Message, error) {
	rows, err := r.store.MessageListByChannelKeyset(ctx, db.MessageListByChannelKeysetParams{
		ChannelID:   db.UUID(channelID),
		CursorID:    db.UUIDPtr(cursorID),
		ResultLimit: limit,
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	messages := make([]*channel.Message, len(rows))
	for i, row := range rows {
		messages[i] = messageFromRow(row)
	}
	return messages, nil
}

func (r *Channel) MessageListPinnedByChannel(ctx context.Context, channelID uuid.UUID) ([]*channel.Message, error) {
	rows, err := r.store.MessageListPinnedByChannel(ctx, db.UUID(channelID))
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	messages := make([]*channel.Message, len(rows))
	for i, row := range rows {
		messages[i] = messageFromRow(row)
	}
	return messages, nil
}

func (r *Channel) MessageUpdateContent(ctx context.Context, id uuid.UUID, content string, editedAt time.Time) (*channel.Message, error) {
	row, err := r.store.MessageUpdateContent(ctx, db.MessageUpdateContentParams{
		ID:       db.UUID(id),
		Content:  db.Text(content),
		EditedAt: db.Timestamptz(editedAt),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	return messageFromRow(row), nil
}

func (r *Channel) MessageSetPinned(ctx context.Context, id uuid.UUID, isPinned bool) error {
	err := r.store.MessageSetPinned(ctx, db.MessageSetPinnedParams{
		ID:       db.UUID(id),
		IsPinned: isPinned,
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

// ============================================================================
// MESSAGE REACTIONS
// ============================================================================

func (r *Channel) ReactionAdd(ctx context.Context, reaction *channel.Reaction) error {
	row, err := r.store.ReactionAdd(ctx, db.ReactionAddParams{
		MessageID: db.UUID(reaction.MessageID()),
		UserID:    db.UUID(reaction.UserID()),
		Emoji:     reaction.Emoji(),
		CreatedAt: db.Timestamptz(reaction.CreatedAt()),
	})
	if err != nil {
		return db.NewError(err, db.EntityMessageReaction)
	}

	*reaction = *reactionFromRow(row)
	return nil
}

func (r *Channel) ReactionRemove(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
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

func (r *Channel) ReactionListByMessage(ctx context.Context, messageID uuid.UUID) ([]*channel.Reaction, error) {
	rows, err := r.store.ReactionListByMessage(ctx, db.UUID(messageID))
	if err != nil {
		return nil, db.NewError(err, db.EntityMessageReaction)
	}

	reactions := make([]*channel.Reaction, len(rows))
	for i, row := range rows {
		reactions[i] = reactionFromRow(row)
	}
	return reactions, nil
}

func (r *Channel) ReactionSummarizeByMessage(ctx context.Context, messageID uuid.UUID) ([]channel.ReactionSummary, error) {
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
// Internal Reconstitution Helpers
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

func sidebarItemFromRow(row db.ChannelMemberListByUserRow) *channel.UserSidebarItem {
	return channel.ReconstituteSidebarItem(
		uuid.UUID(row.ChannelID.Bytes),
		uuid.UUID(row.UserID.Bytes),
		row.JoinedAt.Time.UTC(),
		db.UUIDPtrFromDB(row.LastReadMessageID),
		row.MentionCount,
		channel.Type(row.ChannelType),
		db.UUIDPtrFromDB(row.ChannelOwnerID),
		db.StringPtr(row.ChannelName),
		db.StringPtr(row.ChannelIconUrl),
		db.UUIDPtrFromDB(row.ChannelLastMessageID),
		row.ChannelUpdatedAt.Time.UTC(),
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

func reactionFromRow(row db.MessageReaction) *channel.Reaction {
	return channel.ReconstituteReaction(
		uuid.UUID(row.MessageID.Bytes),
		uuid.UUID(row.UserID.Bytes),
		row.Emoji,
		row.CreatedAt.Time.UTC(),
	)
}
