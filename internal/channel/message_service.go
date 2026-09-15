package channel

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/user"
	"context"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type MessageService struct {
	cache             MessageCache
	repo              MessageRepository
	cachedRepo        CachedMessageRepository
	channelCache      ChannelCache
	channelRepo       ChannelRepository
	cachedChannelRepo CachedChannelRepository
	memberRepo        MemberRepository
	cachedMemberRepo  CachedMemberRepository
	reactionRepo      ReactionRepository
	userRepo          UserRepository
	cachedUserRepo    CachedUserRepository
	outboxRepo        OutboxRepository
	tx                TX
}

func NewMessageService(
	repo MessageRepository,
	cachedRepo CachedMessageRepository,
	channelRepo ChannelRepository,
	cachedChannelRepo CachedChannelRepository,
	memberRepo MemberRepository,
	cachedMemberRepo CachedMemberRepository,
	reactionRepo ReactionRepository,
	userRepo UserRepository,
	cachedUserRepo CachedUserRepository,
	outboxRepo OutboxRepository,
	tx TX,
) *MessageService {
	return &MessageService{
		repo:              repo,
		cachedRepo:        cachedRepo,
		channelRepo:       channelRepo,
		cachedChannelRepo: cachedChannelRepo,
		memberRepo:        memberRepo,
		cachedMemberRepo:  cachedMemberRepo,
		reactionRepo:      reactionRepo,
		userRepo:          userRepo,
		cachedUserRepo:    cachedUserRepo,
		outboxRepo:        outboxRepo,
		tx:                tx,
	}
}

// Create generates a new message and related channel + member side effects.
func (s *MessageService) Create(
	ctx context.Context,
	authorID,
	sessionID,
	channelID uuid.UUID,
	content *string,
	replyToMsgID *uuid.UUID,
	fwdMsgID *uuid.UUID,
	fwdChannelID *uuid.UUID,
) (*Message, error) {
	hasReply := replyToMsgID != nil
	hasFwdMsg := fwdMsgID != nil
	hasFwdChan := fwdChannelID != nil

	if err := validateReply(hasReply, hasFwdMsg, hasFwdChan); err != nil {
		return nil, err
	}

	if err := validateForward(hasFwdMsg, hasFwdChan); err != nil {
		return nil, err
	}

	var mems []*Member

	g, ctxGrp := errgroup.WithContext(ctx)

	g.Go(func() error {
		var gErr error
		mems, gErr = s.getValidMemberships(ctxGrp, channelID, authorID)
		if gErr != nil {
			return gErr
		}
		return nil
	})

	if hasReply {
		g.Go(func() error {
			parentMsg, err := s.cachedRepo.Get(ctxGrp, *replyToMsgID)
			if err != nil {
				return err
			}
			if parentMsg.ChannelID != channelID {
				return ErrMessageReplyDifferentChannel()
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	memberIDs := getMemberIDs(mems)

	author, err := s.cachedUserRepo.Get(ctx, authorID)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	msg, err := NewMessage(
		channelID,
		authorID,
		*content,
		replyToMsgID,
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

		_, err = s.channelRepo.UpdateLastMessage(txCtx, ch.ID, &savedMsg.ID, &savedMsg.CreatedAt, now)
		if err != nil {
			return err
		}

		_, err = s.memberRepo.UpdateLastReadMessage(
			txCtx,
			channelID,
			authorID,
			&msg.ID,
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

		payload := EventMessageCreatedPayload{
			ExcludeSessionID: sessionID,
			Message:          savedMsg,
			Author:           author,
			MemberIDs:        memberIDs,
		}

		return s.outboxRepo.Publish(
			txCtx,
			EventMessageCreated,
			payload,
			now,
		)
	})
	if err != nil {
		return nil, err
	}

	_ = s.cache.Set(ctx, savedMsg)
	_ = s.channelCache.Delete(ctx, channelID)
	_ = s.channelCache.InvalidateMembers(ctx, channelID)

	return savedMsg, nil
}

// ListAround fetches messages directly before and after msgCursorID.
func (s *MessageService) ListAround(
	ctx context.Context,
	actorID, channelID, msgCursorID uuid.UUID,
) (*GetMessageViewsResult, bool, bool, error) {
	if err := s.validateMembership(ctx, channelID, actorID); err != nil {
		return nil, false, false, err
	}

	messages, hasMoreBefore, hasMoreAfter, err := s.cachedRepo.ListAroundByChannelID(
		ctx,
		channelID,
		msgCursorID,
		MessageListBeforeLimit,
		MessageListAfterLimit,
	)
	if err != nil {
		return nil, false, false, err
	}

	views, err := s.getMessageViews(ctx, actorID, messages)
	if err != nil {
		return nil, false, false, err
	}

	return views, hasMoreBefore, hasMoreAfter, err
}

// ListBefore fetches messages directly before msgCursorID using the cached repository.
func (s *MessageService) ListBefore(
	ctx context.Context,
	actorID, channelID, msgCursorID uuid.UUID,
) (*GetMessageViewsResult, bool, error) {
	if err := s.validateMembership(ctx, channelID, actorID); err != nil {
		return nil, false, err
	}

	messages, hasMoreBefore, err := s.cachedRepo.ListBeforeByChannelID(
		ctx,
		channelID,
		msgCursorID,
		MessageListLimit,
	)
	if err != nil {
		return nil, false, err
	}

	views, err := s.getMessageViews(ctx, actorID, messages)
	if err != nil {
		return nil, false, err
	}

	return views, hasMoreBefore, nil
}

// ListAfter fetches messages directly after msgCursorID using the cached repository.
func (s *MessageService) ListAfter(
	ctx context.Context,
	actorID, channelID, msgCursorID uuid.UUID,
) (*GetMessageViewsResult, bool, error) {
	if err := s.validateMembership(ctx, channelID, actorID); err != nil {
		return nil, false, err
	}

	messages, hasMoreAfter, err := s.cachedRepo.ListAfterByChannelID(
		ctx,
		channelID,
		msgCursorID,
		MessageListLimit,
	)
	if err != nil {
		return nil, false, err
	}

	views, err := s.getMessageViews(ctx, actorID, messages)
	if err != nil {
		return nil, false, err
	}

	return views, hasMoreAfter, nil
}

// ListPinned fetches pinned messages for a channel
func (s *MessageService) ListPinned(
	ctx context.Context,
	actorID, channelID uuid.UUID,
	msgCursorID *uuid.UUID,
	cursorPinnedAt *time.Time,
) ([]*Message, map[uuid.UUID]*user.User, bool, error) {
	if err := s.validateMembership(ctx, channelID, actorID); err != nil {
		return nil, nil, false, err
	}

	messages, hasMoreBefore, err := s.repo.ListPinnedByChannelID(
		ctx,
		channelID,
		msgCursorID,
		cursorPinnedAt,
		MessageListLimit,
	)
	if err != nil {
		return nil, nil, false, err
	}

	if len(messages) == 0 {
		return nil, nil, false, nil
	}

	_, authorIDs := getMessageIDs(messages)

	userMap, err := s.cachedUserRepo.GetBatch(ctx, authorIDs)
	if err != nil {
		return nil, nil, false, err
	}

	sortPinnedMessages(messages)

	return messages, userMap, hasMoreBefore, nil
}

// UpdateContent updates an author's message content.
func (s *MessageService) UpdateContent(
	ctx context.Context,
	actorID, sessionID, channelID, messageID uuid.UUID,
	content string,
) (*Message, error) {
	if len(content) == 0 {
		return nil, ErrMessageContentMinLength()
	}

	msg, mems, err := s.prepareUpdate(ctx, actorID, channelID, messageID)
	if err != nil {
		return nil, err
	}

	if msg.AuthorID != &actorID {
		return nil, ErrMessageNotAuthor()
	}

	var updatedMsg *Message
	now := time.Now()
	memIDs := getMemberIDs(mems)

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		updatedMsg, err = s.repo.UpdateContent(txCtx, messageID, content, now, now)
		if err != nil {
			return err
		}

		payload := EventMessageUpdatedPayload{
			ExcludeSessionID: sessionID,
			MessageID:        updatedMsg.ID,
			MessageContent:   updatedMsg.Content,
			MemberIDs:        memIDs,
		}

		return s.outboxRepo.Publish(
			txCtx,
			EventMessageUpdated,
			payload,
			now,
		)
	})
	if err != nil {
		return nil, err
	}

	_ = s.cache.Delete(ctx, channelID, messageID)

	return updatedMsg, nil
}

// UpdatePinnedAt pins or unpins a message in a channel.
func (s *MessageService) UpdatePinnedAt(
	ctx context.Context,
	actorID, sessionID, channelID, messageID uuid.UUID,
	isPinned bool,
) (*Message, error) {
	_, mems, err := s.prepareUpdate(ctx, actorID, channelID, messageID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	memIDs := getMemberIDs(mems)

	var pinnedAt *time.Time
	if isPinned {
		pinnedAt = &now
	}

	var updatedMsg *Message
	var savedSysMsg *Message

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		updatedMsg, err = s.repo.UpdatePinnedAt(txCtx, messageID, pinnedAt, now)
		if err != nil {
			return err
		}

		if isPinned {
			sysMsg, err := NewMessagePin(
				channelID,
				&actorID,
				updatedMsg.ID,
				now,
			)
			if err != nil {
				return err
			}

			savedSysMsg, err = s.repo.Create(txCtx, sysMsg)
			if err != nil {
				return err
			}

			_, err = s.channelRepo.UpdateLastMessage(txCtx, channelID, &savedSysMsg.ID, &now, now)
			if err != nil {
				return err
			}
		}

		payload := EventMessageUpdatedPayload{
			MemberIDs:        memIDs,
			ExcludeSessionID: sessionID,
			MessageID:        messageID,
			MessagePinnedAt:  pinnedAt,
			MessageUpdatedAt: updatedMsg.UpdatedAt,
			SystemMessage:    savedSysMsg,
		}

		return s.outboxRepo.Publish(txCtx, EventMessageUpdated, payload, now)
	})
	if err != nil {
		return nil, err
	}

	_ = s.cache.Delete(ctx, channelID, messageID)

	if savedSysMsg != nil {
		_ = s.cache.Set(ctx, savedSysMsg)
		_ = s.channelCache.Delete(ctx, channelID)
	}

	return updatedMsg, nil
}

// Delete deletes a message belonging to the actor and triggers side effects.
func (s *MessageService) Delete(
	ctx context.Context,
	actorID, sessionID, channelID, messageID uuid.UUID,
) error {
	msg, mems, err := s.prepareUpdate(ctx, actorID, channelID, messageID)
	if err != nil {
		return err
	}

	if msg.AuthorID != &actorID {
		return ErrMessageNotAuthor()
	}

	now := time.Now()
	memIDs := getMemberIDs(mems)

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if txErr := s.repo.Delete(txCtx, messageID); txErr != nil {
			return txErr
		}

		payload := EventMessageDeletedPayload{
			MemberIDs:        memIDs,
			ExcludeSessionID: sessionID,
			MessageID:        messageID,
			MessageDeletedAt: now,
		}

		return s.outboxRepo.Publish(txCtx, EventMessageDeleted, payload, now)
	})
	if err != nil {
		return err
	}

	_ = s.cache.Delete(ctx, channelID, messageID)

	return nil
}

// ToggleReaction adds or removes a user's reaction on a message.
func (s *MessageService) ToggleReaction(
	ctx context.Context,
	actorID, sessionID, channelID, messageID uuid.UUID,
	emoji string,
) (*EmojiCount, error) {
	_, mems, err := s.prepareUpdate(ctx, actorID, channelID, messageID)
	if err != nil {
		return nil, err
	}

	var (
		willBeReacted bool
		updatedCount  int
	)

	now := time.Now()
	memberIDs := getMemberIDs(mems)

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		existingRx, txErr := s.reactionRepo.Get(txCtx, messageID, actorID, emoji)
		if txErr != nil && !errs.IsNotFound(txErr) {
			return txErr
		}

		wasReacted := existingRx != nil
		willBeReacted = !wasReacted

		if wasReacted {
			if txErr := s.reactionRepo.Delete(txCtx, messageID, actorID, emoji); txErr != nil {
				return txErr
			}
		} else {
			rx := ReconstituteReaction(messageID, actorID, emoji, now)
			if _, txErr := s.reactionRepo.Create(txCtx, rx); txErr != nil {
				return txErr
			}
		}

		updatedCount, txErr = s.reactionRepo.CountByEmoji(txCtx, messageID, emoji)
		if txErr != nil {
			return txErr
		}

		// Broadcast neutral Reacted: false so clients don't overwrite user-specific state
		broadcastEmojiCount := EmojiCount{
			Emoji:   emoji,
			Count:   updatedCount,
			Reacted: false,
		}

		payload := EventReactionToggledPayload{
			MemberIDs:        memberIDs,
			ExcludeSessionID: sessionID,
			ActorID:          actorID,
			MessageID:        messageID,
			EmojiCount:       broadcastEmojiCount,
			ToggledAt:        now,
		}

		return s.outboxRepo.Publish(txCtx, EventReactionToggled, payload, now)
	})
	if err != nil {
		return nil, err
	}

	return &EmojiCount{
		Emoji:   emoji,
		Count:   updatedCount,
		Reacted: willBeReacted,
	}, nil
}

func (s *MessageService) validateMembership(ctx context.Context, channelID, userID uuid.UUID) error {
	_, err := s.cachedMemberRepo.Get(ctx, channelID, userID)
	if err != nil {
		if errs.IsNotFound(err) {
			return errs.PermissionDenied("You are not a member of this channel.")
		}
		return err
	}
	return nil
}

func (s *MessageService) getValidMemberships(ctx context.Context, channelID, authorID uuid.UUID) ([]*Member, error) {
	mems, err := s.cachedMemberRepo.GetBatchByChannelID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	_, err = validateMembership(authorID, mems)
	if err != nil {
		return nil, err
	}

	return mems, nil
}

type GetMessageViewsResult struct {
	messages          []*Message
	users             map[uuid.UUID]*user.User
	reactionSummaries map[uuid.UUID]*ReactionSummary
}

func (s *MessageService) getMessageViews(ctx context.Context, actorID uuid.UUID, messages []*Message) (*GetMessageViewsResult, error) {
	if len(messages) == 0 {
		return nil, nil
	}

	msgIDs, authorIDs := getMessageIDs(messages)

	var (
		reactionSummaryMap map[uuid.UUID]*ReactionSummary
		userMap            map[uuid.UUID]*user.User
	)

	g, ctxGrp := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		reactionSummaryMap, err = s.reactionRepo.GetBatchSummaryByMessageIDs(ctxGrp, actorID, msgIDs)
		return err
	})

	g.Go(func() error {
		var err error
		userMap, err = s.cachedUserRepo.GetBatch(ctxGrp, authorIDs)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	sortMessages(messages)

	return &GetMessageViewsResult{messages, userMap, reactionSummaryMap}, nil
}

func (s *MessageService) prepareUpdate(ctx context.Context, actorID, channelID, msgID uuid.UUID) (*Message, []*Member, error) {
	var msg *Message
	var mems []*Member

	g, ctxGrp := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		msg, err = s.cachedRepo.Get(ctxGrp, msgID)
		return err
	})

	g.Go(func() error {
		var err error
		mems, err = s.getValidMemberships(ctxGrp, channelID, actorID)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, nil, err
	}

	if msg.ChannelID != channelID {
		return nil, nil, ErrMessageNotFoundInChannel()
	}

	return msg, mems, nil
}
