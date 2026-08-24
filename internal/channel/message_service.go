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
	hasReply := rawReplyToMsgID != nil
	hasFwdMsg := rawFwdMsgID != nil
	hasFwdChan := rawFwdChannelID != nil

	if err := validateReply(hasReply, hasFwdMsg, hasFwdChan); err != nil {
		return nil, err
	}

	if err := validateForward(hasFwdMsg, hasFwdChan); err != nil {
		return nil, err
	}

	authorID, channelID, err := validateIDs(rawAuthorID, rawChannelID)
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

	g.Go(func() error {
		_, err := s.memberRepo.Require(ctxGrp, channelID, authorID)
		if err != nil {
			return err
		}
		return nil
	})

	if hasReply {
		g.Go(func() error {
			parentMsg, err := s.repo.Get(ctxGrp, replyToID)
			if err != nil {
				return err
			}
			if !parentMsg.ChannelID().Equals(channelID) {
				return ErrMessageReplyDifferentChannel()
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	author, err := s.userRepo.Get(ctx, authorID)
	if err != nil {
		return nil, err
	}

	now := fields.Now()

	msg, err := NewMessage(
		channelID,
		authorID,
		content,
		replyToID,
		fwdMsgID,
		fwdChannelID,
		now,
	)
	if err != nil {
		return nil, err
	}

	var savedMsg *Message

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		ch, err := s.channelRepo.GetForUpdate(txCtx, channelID)
		if err != nil {
			return err
		}

		savedMsg, err = s.repo.Create(txCtx, msg)
		if err != nil {
			return err
		}

		_, err = s.channelRepo.UpdateLastMessage(txCtx, ch.ID(), savedMsg.ID(), now, now)
		if err != nil {
			return err
		}

		_, err = s.memberRepo.UpdateLastReadMessage(
			txCtx,
			channelID,
			authorID,
			msg.ID(),
			now,
			now,
			ptr.To(0),
		)
		if err != nil {
			return err
		}

		err = s.memberRepo.IncrementPeersMentionCountByChannelID(txCtx, channelID, authorID, 1, now)
		if err != nil {
			return err
		}

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

	return hydrateMessageView(savedMsg, author, nil), nil
}

// ListAround fetches messages directly before and after rawMsgCursorID.
func (s *MessageService) ListAround(
	ctx context.Context,
	rawActorID, rawChannelID, rawMsgCursorID uuid.UUID,
) ([]MessageView, error) {
	actorID, channelID, msgCursorID, err := s.validateParams(ctx, rawActorID, rawChannelID, rawMsgCursorID)
	if err != nil {
		return nil, err
	}

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

	return s.getMessageViews(ctx, actorID, messages)
}

// ListBefore fetches messages directly before rawMsgCursorID.
func (s *MessageService) ListBefore(
	ctx context.Context,
	rawActorID, rawChannelID, rawMsgCursorID uuid.UUID,
) ([]MessageView, error) {
	actorID, channelID, msgCursorID, err := s.validateParams(ctx, rawActorID, rawChannelID, rawMsgCursorID)
	if err != nil {
		return nil, err
	}

	messages, err := s.repo.ListBeforeByChannelID(ctx, channelID, msgCursorID, MessageListLimit)
	if err != nil {
		return nil, err
	}

	return s.getMessageViews(ctx, actorID, messages)
}

// ListBefore fetches messages directly before rawMsgCursorID.
func (s *MessageService) ListAfter(
	ctx context.Context,
	rawActorID, rawChannelID, rawMsgCursorID uuid.UUID,
) ([]MessageView, error) {
	actorID, channelID, msgCursorID, err := s.validateParams(ctx, rawActorID, rawChannelID, rawMsgCursorID)
	if err != nil {
		return nil, err
	}

	messages, err := s.repo.ListAfterByChannelID(ctx, channelID, msgCursorID, MessageListLimit)
	if err != nil {
		return nil, err
	}

	return s.getMessageViews(ctx, actorID, messages)
}

// ListPinned fetches pinned messages for a channel
func (s *MessageService) ListPinned(
	ctx context.Context,
	rawActorID, rawChannelID uuid.UUID,
	rawMsgCursorID *uuid.UUID,
	rawCursorPinnedAt *time.Time,
) ([]MessagePinnedView, error) {
	_, channelID, msgCursorID, err := s.validateParams(ctx, rawActorID, rawChannelID, ptr.From(rawMsgCursorID))
	if err != nil {
		return nil, err
	}

	cursorPinnedAt := fields.NewTimestamp(ptr.From(rawCursorPinnedAt))

	messages, err := s.repo.ListPinnedByChannelID(
		ctx,
		channelID,
		msgCursorID,
		cursorPinnedAt,
		MessageListLimit,
	)
	if err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		return []MessagePinnedView{}, nil
	}

	_, authorIDs := getMessageIDs(messages)

	userMap, err := s.userRepo.GetBatch(ctx, authorIDs)
	if err != nil {
		return nil, err
	}

	sortPinnedMessages(messages)

	return hydrateMessagePinnedViews(messages, userMap), nil
}

// UpdateContent updates an author's message content.
func (s *MessageService) UpdateContent(
	ctx context.Context,
	rawActorID, rawChannelID, rawMessageID uuid.UUID,
	rawContent string,
) (*Message, error) {
	actorID, channelID, messageID, err := validateMessageIDs(rawActorID, rawChannelID, rawMessageID)
	if err != nil {
		return nil, err
	}

	content, err := ParseMessageContent(rawContent)
	if err != nil {
		return nil, err
	}

	if content.Len() == 0 {
		return nil, ErrMessageContentMinLength()
	}

	msg, err := s.prepareUpdate(ctx, actorID, channelID, messageID)
	if err != nil {
		return nil, err
	}

	if !msg.AuthorID().Equals(actorID) {
		return nil, ErrMessageNotAuthor()
	}

	var updatedMsg *Message

	now := fields.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		updatedMsg, err = s.repo.UpdateContent(txCtx, messageID, content, now, now)
		if err != nil {
			return err
		}

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
	actorID, channelID, messageID, err := validateMessageIDs(rawActorID, rawChannelID, rawMessageID)
	if err != nil {
		return nil, err
	}

	msg, err := s.prepareUpdate(ctx, actorID, channelID, messageID)
	if err != nil {
		return nil, err
	}

	pinnedAt := fields.Timestamp{}

	now := fields.Now()

	if isPinned {
		pinnedAt = now
	}

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		msg, err = s.repo.UpdatePinnedAt(txCtx, messageID, pinnedAt, now)
		if err != nil {
			return err
		}

		if isPinned {
			sysMsg, err := NewMessagePin(
				channelID,
				actorID,
				msg.ID(),
				now,
			)
			if err != nil {
				return err
			}

			_, err = s.repo.Create(txCtx, sysMsg)
			if err != nil {
				return err
			}
		}

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
	actorID, channelID, messageID, err := validateMessageIDs(rawActorID, rawChannelID, rawMessageID)
	if err != nil {
		return err
	}

	msg, err := s.prepareUpdate(ctx, actorID, channelID, messageID)
	if err != nil {
		return err
	}

	if !msg.AuthorID().Equals(actorID) {
		return ErrMessageNotAuthorizedToDelete()
	}

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if txErr := s.repo.Delete(txCtx, messageID); txErr != nil {
			return txErr
		}

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
	actorID, channelID, messageID, err := validateMessageIDs(rawActorID, rawChannelID, rawMessageID)
	if err != nil {
		return nil, err
	}

	emoji, err := ParseReactionEmoji(rawEmoji)
	if err != nil {
		return nil, err
	}

	_, err = s.prepareUpdate(ctx, actorID, channelID, messageID)
	if err != nil {
		return nil, err
	}

	var (
		willBeReacted bool
		updatedCount  int
	)

	now := fields.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
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
			if _, txErr := s.reactionRepo.Create(txCtx, ReconstituteReaction(messageID, actorID, emoji, now)); txErr != nil {
				return txErr
			}
		}

		updatedCount, txErr = s.reactionRepo.CountByEmoji(txCtx, messageID, emoji)
		if txErr != nil {
			return txErr
		}

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

func (s *MessageService) getMessageViews(ctx context.Context, actorID fields.ID, messages []*Message) ([]MessageView, error) {
	if len(messages) == 0 {
		return []MessageView{}, nil
	}

	msgIDs, authorIDs := getMessageIDs(messages)

	var (
		reactionSummaryMap map[fields.ID]*ReactionSummary
		userMap            map[fields.ID]*user.User
	)

	g, ctxGrp := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		reactionSummaryMap, err = s.reactionRepo.GetBatchSummaryByMessageIDs(ctxGrp, actorID, msgIDs)
		return err
	})

	g.Go(func() error {
		var err error
		userMap, err = s.userRepo.GetBatch(ctxGrp, authorIDs)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	sortMessages(messages)

	return hydrateMessageViews(messages, userMap, reactionSummaryMap), nil
}

func (s *MessageService) prepareUpdate(ctx context.Context, actorID, channelID, msgID fields.ID) (*Message, error) {
	var msg *Message

	g, ctxGrp := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		msg, err = s.repo.Get(ctxGrp, msgID)
		return err
	})

	g.Go(func() error {
		_, err := s.memberRepo.Get(ctxGrp, channelID, actorID)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	if !msg.ChannelID().Equals(channelID) {
		return nil, ErrMessageNotFoundInChannel()
	}

	return msg, nil
}

func (s *MessageService) validateParams(
	ctx context.Context,
	rawActorID, rawChannelID, rawMsgID uuid.UUID,
) (actorID, channelID, msgID fields.ID, err error) {
	actorID, channelID, msgID, err = validateMessageIDs(rawActorID, rawChannelID, rawMsgID)
	if err != nil {
		return fields.ID{}, fields.ID{}, fields.ID{}, err
	}

	if _, err = s.memberRepo.Require(ctx, channelID, actorID); err != nil {
		return fields.ID{}, fields.ID{}, fields.ID{}, err
	}

	return actorID, channelID, msgID, nil
}
