package channel

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/user"
	"context"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type MessageService struct {
	repo         MessageRepository
	channelRepo  ChannelRepository
	memberRepo   MemberRepository
	userRepo     UserRepository
	userCache    UserCache
	reactionRepo ReactionRepository
	outboxRepo   OutboxRepository
	tx           TX
}

func NewMessageService(
	repo MessageRepository,
	channelRepo ChannelRepository,
	memberRepo MemberRepository,
	reactionRepo ReactionRepository,
	userRepo UserRepository,
	userCache UserCache,
	tx TX,
) *MessageService {
	return &MessageService{
		repo:         repo,
		channelRepo:  channelRepo,
		memberRepo:   memberRepo,
		reactionRepo: reactionRepo,
		userRepo:     userRepo,
		userCache:    userCache,
		tx:           tx,
	}
}

// Create generates a new message and related channel and member side effects.
func (s *MessageService) Create(
	ctx context.Context,
	rawAuthorID,
	rawChannelID uuid.UUID,
	rawContent *string,
	rawReplyToMsgID *uuid.UUID,
	rawFwdMsgID *uuid.UUID,
	rawFwdChannelID *uuid.UUID,
) (*MessageView, error) {
	// Validate
	hasReply := rawReplyToMsgID != nil
	hasFwdMsg := rawFwdMsgID != nil
	hasFwdChan := rawFwdChannelID != nil

	if hasReply && (hasFwdMsg || hasFwdChan) {
		return nil, errs.InvalidArgument("Cannot reply to a message and forward a message at the same time.")
	}

	if hasFwdMsg != hasFwdChan {
		return nil, errs.InvalidArgument("Forwarded message ID and forwarded channel ID must be provided together.")
	}

	authorID, err := fields.ParseRequiredID("author_id", rawAuthorID)
	if err != nil {
		return nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return nil, err
	}

	content, err := ParseMessageContent(ptr.From(rawContent))
	if err != nil {
		return nil, err
	}

	replyToID, err := fields.ParseID(ptr.From(rawReplyToMsgID))
	if err != nil {
		return nil, err
	}

	fwdMsgID, err := fields.ParseID(ptr.From(rawFwdMsgID))
	if err != nil {
		return nil, err
	}

	fwdChannelID, err := fields.ParseID(ptr.From(rawFwdChannelID))
	if err != nil {
		return nil, err
	}

	g, ctxGrp := errgroup.WithContext(ctx)

	// Validate membership
	g.Go(func() error {
		_, err := s.memberRepo.Get(ctxGrp, channelID, authorID)
		if err != nil {
			if errs.IsNotFound(err) {
				return errs.PermissionDenied("You are not a member of this channel.")
			}
			return err
		}
		return nil
	})

	// Validate reply channel id
	if hasReply {
		g.Go(func() error {
			parentMsg, err := s.repo.Get(ctxGrp, replyToID)
			if err != nil {
				return err
			}
			if parentMsg.ChannelID() != channelID {
				return errs.InvalidArgument("Cannot reply to a message in a different channel.")
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Fetch author for hydration
	author, err := s.userRepo.Get(ctx, authorID)
	if err != nil {
		return nil, err
	}

	msgID, err := fields.NewID()
	if err != nil {
		return nil, err
	}

	now := fields.Now()

	msg := ParseMessage(
		msgID,
		channelID,
		authorID,
		NewMessageType(MessageTypeDefault),
		content,
		fields.JSON{},
		replyToID,
		fwdMsgID,
		fwdChannelID,
		fields.Timestamp{},
		now,
		now,
		fields.Timestamp{},
	)

	var savedMsg *Message

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Lock channel
		ch, err := s.channelRepo.GetForUpdate(txCtx, channelID)
		if err != nil {
			return err
		}

		// Create message
		savedMsg, err = s.repo.Create(txCtx, msg)
		if err != nil {
			return err
		}

		// Update channel with last message
		_, err = s.channelRepo.UpdateLastMessage(txCtx, ch.ID(), savedMsg.ID(), now, now)
		if err != nil {
			return err
		}

		// Update author membership last read message
		_, err = s.memberRepo.UpdateLastReadMessage(
			txCtx,
			channelID,
			authorID,
			msgID,
			now,
			now,
			ptr.To(int32(0)),
		)
		if err != nil {
			return err
		}

		// Increment peer mention count
		err = s.memberRepo.IncrementPeersMentionCountByChannelID(txCtx, channelID, authorID, now)
		if err != nil {
			return err
		}

		// Publish outbox event
		_, err = s.outboxRepo.Publish(
			txCtx,
			EventMessageCreated,
			MessageCreatedPayload{},
		)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Hydrate
	view := HydrateMessageView(savedMsg, author, nil)

	return &view, nil
}

// ListAround fetches messages directly before and after rawMsgCursorID.
func (s *MessageService) ListAround(
	ctx context.Context,
	rawActorID, rawChannelID, rawMsgCursorID uuid.UUID,
) ([]MessageView, error) {
	// Validate
	actorID, err := fields.ParseRequiredID("user_id", rawActorID)
	if err != nil {
		return nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return nil, err
	}

	msgCursorID, err := fields.ParseRequiredID("msg_cursor_id", rawMsgCursorID)
	if err != nil {
		return nil, err
	}

	// Validate membership
	_, err = s.memberRepo.Get(ctx, channelID, actorID)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil, errs.PermissionDenied("You are not a member of this channel.")
		}
		return nil, err
	}

	// Fetch messages around the cursor using standard limits
	messages, err := s.repo.ListAroundByChannelID(
		ctx,
		channelID,
		msgCursorID,
		MessageListBeforeLimit,
		MessageListAfterLimit,
	)
	if err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		return []MessageView{}, nil
	}

	// Collect author ids
	rawUserIDs := make([]fields.ID, len(messages))
	for i, msg := range messages {
		rawUserIDs[i] = msg.AuthorID()
	}
	userIDs := fields.DedupeIDs(rawUserIDs)

	// Concurrent fetch of Reaction Summaries and Users
	var (
		reactionSummaryMap map[fields.ID]*ReactionSummary
		userMap            map[fields.ID]*user.User
	)

	g, ctxGrp := errgroup.WithContext(ctx)

	// Fetch reaction summaries
	g.Go(func() error {
		messageIDs := make([]fields.ID, len(messages))
		for i, msg := range messages {
			messageIDs[i] = msg.ID()
		}
		var err error
		reactionSummaryMap, err = s.reactionRepo.GetBatchSummaryByMessageIDs(ctxGrp, actorID, messageIDs)
		return err
	})

	// Fetch users
	g.Go(func() error {
		var err error
		userMap, err = s.userRepo.GetBatch(ctxGrp, userIDs)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Sort messages chronologically
	SortMessages(messages)

	// Hydrate and return
	return HydrateMessageViews(messages, userMap, reactionSummaryMap), nil
}

// ListBefore fetches messages directly before rawMsgCursorID.
func (s *MessageService) ListBefore(
	ctx context.Context,
	rawActorID, rawChannelID, rawMsgCursorID uuid.UUID,
) ([]MessageView, error) {
	// Validate
	actorID, err := fields.ParseRequiredID("user_id", rawActorID)
	if err != nil {
		return nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return nil, err
	}

	msgCursorID, err := fields.ParseRequiredID("msg_cursor_id", rawMsgCursorID)
	if err != nil {
		return nil, err
	}

	// Validate membership
	_, err = s.memberRepo.Get(ctx, channelID, actorID)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil, errs.PermissionDenied("You are not a member of this channel.")
		}
		return nil, err
	}

	// Fetch messages
	messages, err := s.repo.ListBeforeByChannelID(ctx, channelID, msgCursorID, MessageListLimit)
	if err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		return []MessageView{}, nil
	}

	// Collect author ids
	rawUserIDs := make([]fields.ID, len(messages))
	for i, msg := range messages {
		rawUserIDs[i] = msg.AuthorID()
	}
	userIDs := fields.DedupeIDs(rawUserIDs)

	var (
		reactionSummaryMap map[fields.ID]*ReactionSummary
		userMap            map[fields.ID]*user.User
	)

	g, ctxGrp := errgroup.WithContext(ctx)

	// Fetch reaction summaries
	g.Go(func() error {
		messageIDs := make([]fields.ID, len(messages))
		for i, msg := range messages {
			messageIDs[i] = msg.ID()
		}
		var err error
		reactionSummaryMap, err = s.reactionRepo.GetBatchSummaryByMessageIDs(ctxGrp, actorID, messageIDs)
		return err
	})

	// Fetch users
	g.Go(func() error {
		var err error
		userMap, err = s.userRepo.GetBatch(ctxGrp, userIDs)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Sort
	SortMessages(messages)

	// Hydrate
	view := HydrateMessageViews(messages, userMap, reactionSummaryMap)

	return view, nil
}

// ListBefore fetches messages directly before rawMsgCursorID.
func (s *MessageService) ListAfter(
	ctx context.Context,
	rawUserID, rawChannelID, rawMsgCursorID uuid.UUID,
) ([]MessageView, error) {
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return nil, err
	}

	msgCursorID, err := fields.ParseRequiredID("msg_cursor_id", rawMsgCursorID)
	if err != nil {
		return nil, err
	}

	// Validate membership
	_, err = s.memberRepo.Get(ctx, channelID, userID)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil, errs.PermissionDenied("You are not a member of this channel.")
		}
		return nil, err
	}

	// Fetch messages
	messages, err := s.repo.ListAfterByChannelID(ctx, channelID, msgCursorID, MessageListLimit)
	if err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		return []MessageView{}, nil
	}

	// Collect author ids
	rawUserIDs := make([]fields.ID, len(messages))
	for i, msg := range messages {
		rawUserIDs[i] = msg.AuthorID()
	}
	userIDs := fields.DedupeIDs(rawUserIDs)

	// 4. Concurrent fetch of Reactions and Users
	var (
		reactionSummaryMap map[fields.ID]*ReactionSummary
		userMap            map[fields.ID]*user.User
	)

	g, ctxGrp := errgroup.WithContext(ctx)

	// Fetch reaction summaries
	g.Go(func() error {
		messageIDs := make([]fields.ID, len(messages))
		for i, msg := range messages {
			messageIDs[i] = msg.ID()
		}
		var err error
		reactionSummaryMap, err = s.reactionRepo.GetBatchSummaryByMessageIDs(ctxGrp, userID, messageIDs)
		return err
	})

	// Fetch users
	g.Go(func() error {
		var err error
		userMap, err = s.userRepo.GetBatch(ctxGrp, userIDs)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Sort
	SortMessages(messages)

	// Hydrate
	view := HydrateMessageViews(messages, userMap, reactionSummaryMap)

	return view, nil
}

// ListPinned fetches pinned messages for a channel
func (s *ChannelService) ListPinned(
	ctx context.Context,
	rawActorID, rawChannelID uuid.UUID,
	rawCursorID *uuid.UUID,
	rawCursorPinnedAt *time.Time,
) ([]MessagePinnedView, error) {
	// Validate
	if (rawCursorID == nil) != (rawCursorPinnedAt == nil) {
		return nil, errs.InvalidArgument("Both cursor_id and cursor_pinned_at must be provided together.")
	}

	actorID, err := fields.ParseRequiredID("user_id", rawActorID)
	if err != nil {
		return nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return nil, err
	}

	cursorID, err := fields.ParseID(ptr.From(rawCursorID))
	if err != nil {
		return nil, err
	}

	cursorPinnedAt := fields.NewTimestamp(ptr.From(rawCursorPinnedAt))

	// Validate membership
	_, err = s.memberRepo.Get(ctx, channelID, actorID)
	if err != nil {
		return nil, err
	}

	// Fetch messages
	messages, err := s.messageRepo.ListPinnedByChannelID(
		ctx,
		channelID,
		cursorID,
		cursorPinnedAt,
		MessageListLimit,
	)
	if err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		return []MessagePinnedView{}, nil
	}

	// Collect author ids
	rawUserIDs := make([]fields.ID, len(messages))
	for i, msg := range messages {
		rawUserIDs[i] = msg.AuthorID()
	}
	userIDs := fields.DedupeIDs(rawUserIDs)

	// Fetch users
	userMap, err := s.userRepo.GetBatch(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	// Sort
	SortPinnedMessages(messages)

	// Hydrate
	views := HydrateMessagePinnedViews(messages, userMap)

	return views, nil
}

// UpdateContent updates an author's message content.
func (s *MessageService) UpdateContent(
	ctx context.Context,
	rawActorID, rawChannelID, rawMessageID uuid.UUID,
	rawContent string,
) (*Message, error) {
	// Validate
	actorID, err := fields.ParseRequiredID("user_id", rawActorID)
	if err != nil {
		return nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return nil, err
	}

	messageID, err := fields.ParseRequiredID("message_id", rawMessageID)
	if err != nil {
		return nil, err
	}

	content, err := ParseMessageContent(rawContent)
	if err != nil {
		return nil, err
	}

	if content.Len() == 0 {
		return nil, errs.InvalidArgument("Content must have at least 1 character.")
	}

	// Validate message and membership
	var msg *Message

	g, ctxGrp := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		msg, err = s.repo.Get(ctxGrp, messageID)
		return err
	})

	g.Go(func() error {
		_, err := s.memberRepo.Get(ctxGrp, channelID, actorID)
		if err != nil {
			if errs.IsNotFound(err) {
				return errs.PermissionDenied("You are not a member of this channel.")
			}
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	if !msg.ChannelID().Equals(channelID) {
		return nil, errs.NotFound("Message not found in this channel.")
	}

	if !msg.AuthorID().Equals(actorID) {
		return nil, errs.PermissionDenied("Actor is not the author of the message.")
	}

	var updatedMsg *Message

	now := fields.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Update message
		updatedMsg, err = s.repo.UpdateContent(txCtx, messageID, content, now, now)
		if err != nil {
			return err
		}

		// Publish event
		_, err = s.outboxRepo.Publish(
			txCtx,
			EventMessageUpdateContent,
			MessageUpdateContentPayload{},
		)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedMsg, nil
}

// UpdatePinnedAt pins or unpins a message in a channel.
func (s *MessageService) UpdatePinnedAt(
	ctx context.Context,
	rawActorID, rawChannelID, rawMessageID uuid.UUID,
	isPinned bool,
) (*Message, error) {
	actorID, err := fields.ParseRequiredID("user_id", rawActorID)
	if err != nil {
		return nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return nil, err
	}

	messageID, err := fields.ParseRequiredID("message_id", rawMessageID)
	if err != nil {
		return nil, err
	}

	// Message and member validation
	var (
		msg *Message
	)

	g, ctxGrp := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		msg, err = s.repo.Get(ctxGrp, messageID)
		return err
	})

	g.Go(func() error {
		_, err := s.memberRepo.Get(ctxGrp, channelID, actorID)
		if err != nil {
			if errs.IsNotFound(err) {
				return errs.PermissionDenied("You are not a member of this channel.")
			}
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	if !msg.ChannelID().Equals(channelID) {
		return nil, errs.NotFound("Message not found in this channel.")
	}

	now := fields.Now()
	pinnedAt := fields.Timestamp{}
	if isPinned {
		pinnedAt = now
	}

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Update message
		msg, err = s.repo.UpdatePinnedAt(txCtx, messageID, pinnedAt, now)
		if err != nil {
			return err
		}

		// If pinned, create a system message
		if isPinned {
			msgID, err := fields.NewID()
			if err != nil {
				return err
			}

			sysMsg := ParseMessagePin(
				msgID,
				msg.ChannelID(),
				actorID,
				msg.ID(),
				now,
			)

			_, err = s.repo.Create(txCtx, sysMsg)
			if err != nil {
				return err
			}
		}

		// Publish event
		_, err = s.outboxRepo.Publish(
			txCtx,
			EventMessageUpdatePinnedAt,
			MessageUpdatePinnedAtPayload{},
		)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// Delete deletes a message belonging to the actor and triggers side effects.
func (s *MessageService) Delete(
	ctx context.Context,
	rawActorID, rawChannelID, rawMessageID uuid.UUID,
) error {
	// Validate
	actorID, err := fields.ParseRequiredID("user_id", rawActorID)
	if err != nil {
		return err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return err
	}

	messageID, err := fields.ParseRequiredID("message_id", rawMessageID)
	if err != nil {
		return err
	}

	// Message and member validation
	var msg *Message

	g, ctxGrp := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		msg, err = s.repo.Get(ctxGrp, messageID)
		return err
	})

	g.Go(func() error {
		_, err := s.memberRepo.Get(ctxGrp, channelID, actorID)
		if err != nil {
			if errs.IsNotFound(err) {
				return errs.PermissionDenied("You are not a member of this channel.")
			}
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}

	if !msg.ChannelID().Equals(channelID) {
		return errs.NotFound("Message not found in this channel.")
	}

	if !msg.AuthorID().Equals(actorID) {
		return errs.PermissionDenied("Actor is not authorized to delete this message.")
	}

	now := fields.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Delete message
		if txErr := s.repo.Delete(txCtx, messageID); txErr != nil {
			return txErr
		}

		// Count remaining messages
		remainingCount, err := s.repo.CountByChannelID(txCtx, channelID)
		if err != nil {
			return err
		}

		// If no messages left, clear users and members
		if remainingCount == 0 {
			_, err = s.channelRepo.UpdateLastMessage(txCtx, channelID, fields.ID{}, fields.Timestamp{}, now)
			if err != nil {
				return err
			}

			_, err = s.memberRepo.ClearBatchLastReadMessageByChannelID(txCtx, channelID, now)
			if err != nil {
				return err
			}

			return nil
		}

		// Publish event
		_, txErr := s.outboxRepo.Publish(
			txCtx,
			EventMessageDelete,
			MessageDeletePayload{},
		)
		if txErr != nil {
			return txErr
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// ToggleReaction adds or removes a user's reaction on a message.
func (s *MessageService) ToggleReaction(
	ctx context.Context,
	rawActorID, rawChannelID, rawMessageID uuid.UUID,
	rawEmoji string,
) (*EmojiCount, error) {
	// Validate
	actorID, err := fields.ParseRequiredID("user_id", rawActorID)
	if err != nil {
		return nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return nil, err
	}

	messageID, err := fields.ParseRequiredID("message_id", rawMessageID)
	if err != nil {
		return nil, err
	}

	emoji, err := ParseReactionEmoji(rawEmoji)
	if err != nil {
		return nil, err
	}

	// Validate message and membership
	var msg *Message

	g, ctxGrp := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		msg, err = s.repo.Get(ctxGrp, messageID)
		return err
	})

	g.Go(func() error {
		_, err := s.memberRepo.Get(ctxGrp, channelID, actorID)
		if err != nil {
			if errs.IsNotFound(err) {
				return errs.PermissionDenied("You are not a member of this channel.")
			}
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	if !msg.ChannelID().Equals(channelID) {
		return nil, errs.NotFound("Message not found in this channel.")
	}

	var (
		willBeReacted bool
		updatedCount  int
	)

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Get reaction
		existingRx, txErr := s.reactionRepo.Get(txCtx, messageID, actorID, emoji)
		if txErr != nil && !errs.IsNotFound(txErr) {
			return txErr
		}

		wasReacted := existingRx != nil
		willBeReacted = !wasReacted

		// Create / delete
		if wasReacted {
			if txErr := s.reactionRepo.Delete(txCtx, messageID, actorID, emoji); txErr != nil {
				return txErr
			}
		} else {
			rx := ParseReaction(messageID, actorID, emoji, fields.Now())
			if _, txErr := s.reactionRepo.Create(txCtx, rx); txErr != nil {
				return txErr
			}
		}

		// Calculate emoji count
		updatedCount, txErr = s.reactionRepo.CountByEmoji(txCtx, messageID, emoji)
		if txErr != nil {
			return txErr
		}

		// Publish event
		_, txErr = s.outboxRepo.Publish(
			txCtx,
			EventReactionToggle,
			ReactionTogglePayload{},
		)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	return &EmojiCount{
		Emoji:   emoji.String(),
		Count:   updatedCount,
		Reacted: willBeReacted,
	}, nil
}
