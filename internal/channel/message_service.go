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

type MessageRepository interface {
	Create(ctx context.Context, msg *Message) (*Message, error)
	Delete(ctx context.Context, id fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Message, error)
	ListAfterByChannelID(ctx context.Context, channelID fields.ID, msgCursorID fields.ID, limit int32) ([]*Message, error)
	ListAroundByChannelID(ctx context.Context, channelID fields.ID, lastReadMessageID fields.ID, beforeLimit int32, afterLimit int32) ([]*Message, error)
	ListBeforeByChannelID(ctx context.Context, channelID fields.ID, msgCursorID fields.ID, limit int32) ([]*Message, error)
	ListPinnedByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, cursorPinnedAt fields.Timestamp, limit int32) ([]*Message, error)
	UpdateContent(ctx context.Context, id fields.ID, content MessageContent, editedAt fields.Timestamp, updatedAt fields.Timestamp) (*Message, error)
	UpdatePinnedAt(ctx context.Context, id fields.ID, pinnedAt fields.Timestamp, updatedAt fields.Timestamp) (*Message, error)
}

type MessageCache interface {
	Delete(ctx context.Context, key fields.ID) error
	DeleteBatch(ctx context.Context, keys []fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Message, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*Message, []fields.ID, error)
	Set(ctx context.Context, ch *Message) error
	SetBatch(ctx context.Context, channels []*Message) error
}

type MessageService struct {
	repo          MessageRepository
	cache         MessageCache
	channelRepo   ChannelRepository
	channelCache  ChannelCache
	memberRepo    MemberRepository
	memberCache   MemberCache
	userRepo      UserRepository
	userCache     UserCache
	reactionRepo  ReactionRepository
	reactionCache ReactionCache
	outboxRepo    OutboxRepository
	tx            TX
}

func NewMessageService(
	repo MessageRepository,
	cache MessageCache,
	channelRepo ChannelRepository,
	channelCache ChannelCache,
	memberRepo MemberRepository,
	memberCache MemberCache,
	reactionRepo ReactionRepository,
	reactionCache ReactionCache,
	userRepo UserRepository,
	userCache UserCache,
	tx TX,
) *MessageService {
	return &MessageService{
		repo:          repo,
		cache:         cache,
		channelRepo:   channelRepo,
		channelCache:  channelCache,
		memberRepo:    memberRepo,
		memberCache:   memberCache,
		reactionRepo:  reactionRepo,
		reactionCache: reactionCache,
		userRepo:      userRepo,
		userCache:     userCache,
		tx:            tx,
	}
}

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

	replyToID, err := fields.ParseID("reply_to_message_id", *rawReplyToMsgID)
	if err != nil {
		return nil, err
	}

	fwdMsgID, err := fields.ParseID("forwarded_message_id", *rawFwdMsgID)
	if err != nil {
		return nil, err
	}

	fwdChannelID, err := fields.ParseID("forwarded_channel_id", *rawFwdChannelID)
	if err != nil {
		return nil, err
	}

	author, err := s.userRepo.Get(ctx, authorID)
	if err != nil {
		return nil, err
	}

	if hasReply {
		parentMsg, err := s.repo.Get(ctx, replyToID)
		if err != nil {
			return nil, err
		}
		if parentMsg.ChannelID() != channelID {
			return nil, errs.InvalidArgument("Cannot reply to a message in a different channel.")
		}
	}

	now := fields.NewTimestamp(time.Now())
	msgID := fields.NewID(uuid.New()) // TODO: v7

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

	// 5. Execute Write Transaction
	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Get channel for update to serialize state changes
		ch, err := s.channelRepo.GetForUpdate(txCtx, channelID)
		if err != nil {
			return err
		}

		// Create message
		savedMsg, err = s.repo.Create(txCtx, msg)
		if err != nil {
			return err
		}

		// Update channel domain with last message id and last message at
		_, err = s.channelRepo.UpdateLastMessage(txCtx, ch.ID(), savedMsg.ID(), now, now)
		if err != nil {
			return err
		}

		// Update channel member last read message for the author
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

		// Increment members mention count (excluding author)
		err = s.memberRepo.IncrementPeersMentionCount(txCtx, channelID, authorID, now)
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

	// 6. Hydrate and return view (New messages have no reactions yet, pass empty slice/nil)
	view := HydrateMessage(savedMsg, nil, author, authorID)

	return &view, nil
}

// ListBefore fetches older messages when scrolling up
func (s *MessageService) ListBefore(
	ctx context.Context,
	rawActorID, rawChannelID, rawMsgCursorID uuid.UUID,
) ([]MessageView, error) {
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

	// 1. Authorization: Verify user is a member of the channel
	_, err = s.memberRepo.Get(ctx, channelID, actorID)
	if err != nil {
		return nil, err
	}

	// 2. Fetch older messages
	messages, err := s.repo.ListBeforeByChannelID(ctx, channelID, msgCursorID, MessageListLimit)
	if err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		return []MessageView{}, nil
	}

	// 3. Collect author IDs for hydration
	userIDMap := make(map[fields.ID]struct{}, len(messages))
	for _, msg := range messages {
		if id := msg.AuthorID(); id.IsValid() {
			userIDMap[id] = struct{}{}
		}
	}
	userIDs := make([]fields.ID, 0, len(userIDMap))
	for id := range userIDMap {
		userIDs = append(userIDs, id)
	}

	// 4. Concurrent fetch of Reactions and Users
	var (
		reactionMap map[fields.ID][]*Reaction
		userMap     map[fields.ID]*user.User
	)

	g, ctxGrp := errgroup.WithContext(ctx)

	g.Go(func() error {
		messageIDs := make([]fields.ID, len(messages))
		for i, msg := range messages {
			messageIDs[i] = msg.ID()
		}
		var err error
		reactionMap, err = s.reactionRepo.GetBatchByMessageIDs(ctxGrp, messageIDs)
		return err
	})

	g.Go(func() error {
		var err error
		userMap, err = s.userRepo.GetBatch(ctxGrp, userIDs)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// 5. Hydrate and return
	return HydrateMessages(messages, reactionMap, userMap, actorID), nil
}

// ListAfter fetches newer messages when scrolling down or catching up
func (s *MessageService) ListAfter(
	ctx context.Context,
	rawUserID, rawChannelID, rawAfterID uuid.UUID,
) ([]MessageView, error) {
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return nil, err
	}

	msgCursorID, err := fields.ParseRequiredID("msg_cursor_id", rawAfterID)
	if err != nil {
		return nil, err
	}

	// 1. Authorization: Verify user is a member of the channel
	_, err = s.memberRepo.Get(ctx, channelID, userID)
	if err != nil {
		return nil, err
	}

	// 2. Fetch newer messages
	messages, err := s.repo.ListAfterByChannelID(ctx, channelID, msgCursorID, MessageListLimit)
	if err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		return []MessageView{}, nil
	}

	// 3. Collect author IDs for hydration
	userIDMap := make(map[fields.ID]struct{}, len(messages))
	for _, msg := range messages {
		if id := msg.AuthorID(); id.IsValid() {
			userIDMap[id] = struct{}{}
		}
	}
	userIDs := make([]fields.ID, 0, len(userIDMap))
	for id := range userIDMap {
		userIDs = append(userIDs, id)
	}

	// 4. Concurrent fetch of Reactions and Users
	var (
		reactionMap map[fields.ID][]*Reaction
		userMap     map[fields.ID]*user.User
	)

	g, ctxGrp := errgroup.WithContext(ctx)

	g.Go(func() error {
		messageIDs := make([]fields.ID, len(messages))
		for i, msg := range messages {
			messageIDs[i] = msg.ID()
		}
		var err error
		reactionMap, err = s.reactionRepo.GetBatchByMessageIDs(ctxGrp, messageIDs)
		return err
	})

	g.Go(func() error {
		var err error
		userMap, err = s.userRepo.GetBatch(ctxGrp, userIDs)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// 5. Hydrate and return
	return HydrateMessages(messages, reactionMap, userMap, userID), nil
}

// ListPinned fetches pinned messages for a channel
func (s *ChannelService) ListPinned(
	ctx context.Context,
	rawActorID, rawChannelID uuid.UUID,
	rawCursorID *uuid.UUID,
	rawCursorPinnedAt *time.Time,
) ([]MessagePinnedView, error) {
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

	cursorID, err := fields.ParseID("cursor_id", ptr.From(rawCursorID))
	if err != nil {
		return nil, err
	}

	cursorPinnedAt := fields.NewTimestamp(ptr.From(rawCursorPinnedAt))

	// 1. Authorization: Verify user is a member of the channel
	_, err = s.memberRepo.Get(ctx, channelID, actorID)
	if err != nil {
		return nil, err
	}

	// 2. Fetch pinned messages (using a reasonable default limit like 50 or MessageListLimit)
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

	// 3. Collect author IDs for hydration
	userIDMap := make(map[fields.ID]struct{}, len(messages))
	for _, msg := range messages {
		if id := msg.AuthorID(); id.IsValid() {
			userIDMap[id] = struct{}{}
		}
	}
	userIDs := make([]fields.ID, 0, len(userIDMap))
	for id := range userIDMap {
		userIDs = append(userIDs, id)
	}

	// 4. Batch fetch users
	userMap, err := s.userRepo.GetBatch(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	// 5. Hydrate and map to MessagePinnedView
	pinnedViews := make([]MessagePinnedView, 0, len(messages))
	for _, msg := range messages {
		u, ok := userMap[msg.AuthorID()]
		if !ok || u == nil {
			u = &user.User{}
		}

		pinnedViews = append(pinnedViews, MessagePinnedView{
			id:          msg.ID(),
			avatarURL:   u.AvatarURL(),
			displayName: u.DisplayName(),
			content:     msg.Content(),
			createdAt:   msg.CreatedAt(),
		})
	}

	return pinnedViews, nil
}

// UpdateContent updates a message's text if the actor is the author
func (s *MessageService) UpdateContent(
	ctx context.Context,
	rawActorID, rawMessageID uuid.UUID,
	rawContent string,
) (*MessageView, error) {
	actorID, err := fields.ParseRequiredID("user_id", rawActorID)
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

	// 1. Fetch existing message
	msg, err := s.repo.Get(ctx, messageID)
	if err != nil {
		return nil, err
	}

	// 2. Authorization: Ensure actor is the author
	if !msg.AuthorID().Equals(actorID) {
		return nil, errs.PermissionDenied("Actor is not the author of the message.")
	}

	// 3. Authorization: Verify user is still a member of the channel
	_, err = s.memberRepo.Get(ctx, msg.ChannelID(), actorID)
	if err != nil {
		return nil, err
	}

	// 4. Concurrent fetch of Author User profile and Message Reactions for hydration
	var (
		author    *user.User
		reactions []*Reaction
	)

	g, ctxGrp := errgroup.WithContext(ctx)

	// Fetch author user details (actorID == authorID)
	g.Go(func() error {
		var uErr error
		author, uErr = s.userRepo.Get(ctxGrp, actorID)
		return uErr
	})

	// Fetch reactions attached to this message
	g.Go(func() error {
		reactionMap, rErr := s.reactionRepo.GetBatchByMessageIDs(ctxGrp, []fields.ID{messageID})
		if rErr != nil {
			return rErr
		}
		reactions = reactionMap[messageID]
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// 5. Execute Write Transaction
	var updatedMsg *Message

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		now := fields.NewTimestamp(time.Now())
		var txErr error

		// 6. Update message in DB (using txCtx)
		updatedMsg, txErr = s.repo.UpdateContent(txCtx, messageID, content, now, now)
		if txErr != nil {
			return txErr
		}

		// 7. Publish Outbox Event for real-time consumers (using txCtx)
		_, txErr = s.outboxRepo.Publish(
			txCtx,
			EventMessageContentUpdated,
			MessageContentUpdatedPayload{},
		)
		if txErr != nil {
			return txErr
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// 8. Hydrate single message view for response
	view := HydrateMessage(updatedMsg, reactions, author, actorID)
	return &view, nil
}

// Update pinned at

// Delete

// Handle reaction
