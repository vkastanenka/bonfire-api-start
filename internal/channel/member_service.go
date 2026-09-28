package channel

import (
	"bonfire-api/internal/appctx"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/helpers"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type MemberService struct {
	cache              MemberCache
	repo               MemberRepository
	cachedRepo         CachedMemberRepository
	channelCache       ChannelCache
	channelRepo        ChannelRepository
	cachedChannelRepo  CachedChannelRepository
	messageCache       MessageCache
	messageRepo        MessageRepository
	userCache          UserCache
	userRepo           UserRepository
	cachedUserRepo     CachedUserRepository
	presenceCache      PresenceCache
	outboxRepo         OutboxRepository
	relationRepo       RelationRepository
	cachedRelationRepo CachedRelationRepository
	tx                 TX
}

func NewMemberService(
	cache MemberCache,
	repo MemberRepository,
	cachedRepo CachedMemberRepository,
	channelCache ChannelCache,
	channelRepo ChannelRepository,
	cachedChannelRepo CachedChannelRepository,
	messageCache MessageCache,
	messageRepo MessageRepository,
	userCache UserCache,
	userRepo UserRepository,
	cachedUserRepo CachedUserRepository,
	presenceCache PresenceCache,
	outboxRepo OutboxRepository,
	relationRepo RelationRepository,
	cachedRelationRepo CachedRelationRepository,
	tx TX,
) *MemberService {
	return &MemberService{
		cache:              cache,
		repo:               repo,
		cachedRepo:         cachedRepo,
		channelCache:       channelCache,
		channelRepo:        channelRepo,
		cachedChannelRepo:  cachedChannelRepo,
		messageCache:       messageCache,
		messageRepo:        messageRepo,
		userCache:          userCache,
		userRepo:           userRepo,
		cachedUserRepo:     cachedUserRepo,
		presenceCache:      presenceCache,
		outboxRepo:         outboxRepo,
		relationRepo:       relationRepo,
		cachedRelationRepo: cachedRelationRepo,
		tx:                 tx,
	}
}

type AddMembersResult struct {
	MemberIDs []uuid.UUID
	Users     map[uuid.UUID]*user.User
	Presences map[uuid.UUID]presence.Presence
	Messages  []*Message
}

// AddMembers adds members to a channel, creates system notification messages, and broadcasts outbox events.
func (s *MemberService) AddMembers(
	ctx context.Context,
	channelID uuid.UUID,
	newPeerIDs []uuid.UUID,
) (*AddMembersResult, error) {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return nil, err
	}

	if err := validateMinMembers(newPeerIDs); err != nil {
		return nil, err
	}

	if err := validateMaxPeers(newPeerIDs); err != nil {
		return nil, err
	}

	parsedPeerIDs, err := filterRequiredPeerIDs(newPeerIDs, claims.UserID)
	if err != nil {
		return nil, err
	}

	var (
		counts          map[uuid.UUID]int
		friendIDs       []uuid.UUID
		existingMembers []*Member
		ch              *Channel
	)

	g, ctxGrp := errgroup.WithContext(ctx)

	g.Go(func() error {
		var fetchErr error
		counts, fetchErr = s.repo.CountBatchByUserID(ctxGrp, parsedPeerIDs)
		if fetchErr != nil {
			return fetchErr
		}

		for _, peerID := range parsedPeerIDs {
			if counts[peerID] >= MaxUserMemberships {
				return ErrPeerMaxChannelsReached()
			}
		}
		return nil
	})

	g.Go(func() error {
		var fetchErr error
		friendIDs, fetchErr = s.cachedRelationRepo.GetFriendIDs(ctxGrp, claims.UserID)
		if fetchErr != nil {
			return fetchErr
		}

		if len(friendIDs) == 0 {
			return ErrCannotAddNonFriendUserToGroup()
		}

		friendSet := make(map[uuid.UUID]struct{}, len(friendIDs))
		for _, id := range friendIDs {
			friendSet[id] = struct{}{}
		}

		for _, peerID := range parsedPeerIDs {
			if _, isFriend := friendSet[peerID]; !isFriend {
				return ErrCannotAddNonFriendUserToGroup()
			}
		}
		return nil
	})

	g.Go(func() error {
		var fetchErr error
		existingMembers, fetchErr = s.cachedRepo.GetBatchByChannelID(ctxGrp, channelID)
		if fetchErr != nil {
			return fetchErr
		}

		return validateMembership(existingMembers, claims.UserID)
	})

	g.Go(func() error {
		var fetchErr error
		ch, fetchErr = s.cachedChannelRepo.Get(ctxGrp, channelID)
		if fetchErr != nil {
			return fetchErr
		}

		if ch.Type.IsDirect() {
			return ErrCannotAddMembersToDirectChannel()
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	candidateMemberIDs, err := filterNewMemberIDs(existingMembers, newPeerIDs, claims.UserID)
	if err != nil {
		return nil, err
	}

	existingMemberIDs := getMemberIDs(existingMembers)
	allMemberIDs := helpers.DedupeIDs(append(existingMemberIDs, candidateMemberIDs...))

	var (
		allUsers     map[uuid.UUID]*user.User
		allPresences map[uuid.UUID]presence.Presence
	)

	gHydrate, ctxHydrate := errgroup.WithContext(ctx)

	gHydrate.Go(func() error {
		var fetchErr error
		allUsers, fetchErr = s.cachedUserRepo.GetBatch(ctxHydrate, allMemberIDs)
		return fetchErr
	})

	gHydrate.Go(func() error {
		var fetchErr error
		allPresences, fetchErr = s.presenceCache.GetBatch(ctxHydrate, allMemberIDs)
		return fetchErr
	})

	if err := gHydrate.Wait(); err != nil {
		return nil, err
	}

	if allUsers == nil {
		allUsers = make(map[uuid.UUID]*user.User)
	}

	now := time.Now()

	var (
		chLock             *Channel
		systemMessages     []*Message
		allMembers         []*Member
		sortedAllMemberIDs []uuid.UUID
		actualAddedIDs     []uuid.UUID
	)

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var fetchErr error
		chLock, fetchErr = s.channelRepo.GetForUpdate(txCtx, channelID)
		if fetchErr != nil {
			return fetchErr
		}

		if chLock.Type.IsDirect() {
			return ErrCannotAddMembersToDirectChannel()
		}

		txMembers, fetchErr := s.repo.GetBatchByChannelID(txCtx, channelID)
		if fetchErr != nil {
			return fetchErr
		}

		actualAddedIDs, fetchErr = filterNewMemberIDs(txMembers, candidateMemberIDs, claims.UserID)
		if fetchErr != nil {
			return fetchErr
		}

		if len(actualAddedIDs) == 0 {
			return nil
		}

		missingUserIDs := make([]uuid.UUID, 0)
		for _, m := range txMembers {
			if _, exists := allUsers[m.UserID]; !exists {
				missingUserIDs = append(missingUserIDs, m.UserID)
			}
		}
		if len(missingUserIDs) > 0 {
			fetchedUsers, fetchErr := s.cachedUserRepo.GetBatch(txCtx, missingUserIDs)
			if fetchErr != nil {
				return fetchErr
			}
			for k, v := range fetchedUsers {
				allUsers[k] = v
			}
		}

		persistedMembers := NewPeers(chLock.ID, actualAddedIDs, now)
		if persistedMembers, fetchErr = s.repo.CreateBatch(txCtx, persistedMembers); fetchErr != nil {
			return fetchErr
		}

		systemMessages, fetchErr = buildAddMembersSystemMessages(chLock.ID, claims.UserID, actualAddedIDs, now)
		if fetchErr != nil {
			return fetchErr
		}

		systemMessages, fetchErr = s.messageRepo.CreateBatch(txCtx, systemMessages)
		if fetchErr != nil {
			return fetchErr
		}

		updatedPeers, fetchErr := s.repo.IncrementPeersMentionCountByChannelID(txCtx, chLock.ID, claims.UserID, len(systemMessages), now)
		if fetchErr != nil {
			return fetchErr
		}

		updatedPeersMap := make(map[uuid.UUID]*Member, len(updatedPeers))
		for _, p := range updatedPeers {
			updatedPeersMap[p.UserID] = p
		}

		combined := append(txMembers, persistedMembers...)
		allMembers = make([]*Member, 0, len(combined))
		for _, m := range combined {
			if updated, exists := updatedPeersMap[m.UserID]; exists {
				allMembers = append(allMembers, updated)
			} else {
				allMembers = append(allMembers, m)
			}
		}

		sortMembers(allMembers, allUsers)
		sortedAllMemberIDs = getMemberIDs(allMembers)
		sortMessages(systemMessages)

		payload := EventMembersAddedPayload{
			Channel:        ParseChannelView(chLock),
			Members:        ParseMemberViewsMap(updatedPeers),
			MemberIDs:      sortedAllMemberIDs,
			Users:          user.ParseSummariesMap(allUsers),
			Presences:      allPresences,
			SystemMessages: ParseMessageViews(systemMessages),
			CreatedAt:      now,
		}

		event, createErr := outbox.New(
			claims.UserID,
			claims.SessionID,
			appctx.GetTraceID(txCtx),
			sortedAllMemberIDs,
			EventMembersAdded,
			payload,
			now,
		)
		if createErr != nil {
			return createErr
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	if len(actualAddedIDs) == 0 {
		return &AddMembersResult{
			MemberIDs: sortedAllMemberIDs,
			Users:     map[uuid.UUID]*user.User{},
			Presences: map[uuid.UUID]presence.Presence{},
			Messages:  []*Message{},
		}, nil
	}

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.cache.InvalidateChannel(cacheCtx, channelID); err != nil {
		slog.ErrorContext(cacheCtx, "failed to invalidate channel members cache",
			slog.String("channel_id", channelID.String()),
			slog.String("error", err.Error()),
		)
	}

	if err := s.messageCache.SetBatch(cacheCtx, chLock.ID, systemMessages); err != nil {
		slog.ErrorContext(cacheCtx, "failed to seed system messages into cache",
			slog.String("channel_id", chLock.ID.String()),
			slog.Int("count", len(systemMessages)),
			slog.Any("error", err.Error()),
		)
	}

	addedUsers := make(map[uuid.UUID]*user.User, len(actualAddedIDs))
	addedPresences := make(map[uuid.UUID]presence.Presence, len(actualAddedIDs))
	for _, id := range actualAddedIDs {
		if u, ok := allUsers[id]; ok {
			addedUsers[id] = u
		}
		if p, ok := allPresences[id]; ok {
			addedPresences[id] = p
		}
	}

	return &AddMembersResult{
		MemberIDs: sortedAllMemberIDs,
		Users:     addedUsers,
		Presences: addedPresences,
		Messages:  systemMessages,
	}, nil
}

func buildAddMembersSystemMessages(
	channelID uuid.UUID,
	actorID uuid.UUID,
	newMemberIDs []uuid.UUID,
	now time.Time,
) ([]*Message, error) {
	systemMessages := make([]*Message, len(newMemberIDs))
	msgTime := now

	for i, addedUserID := range newMemberIDs {
		msg, err := NewMessageMemberAdd(channelID, actorID, addedUserID, msgTime)
		if err != nil {
			return nil, err
		}
		systemMessages[i] = msg
		msgTime = msgTime.Add(time.Microsecond)
	}

	return systemMessages, nil
}

// CloseDirect updates the visibility of a channel membership to false.
func (s *MemberService) CloseDirect(ctx context.Context, channelID uuid.UUID) error {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return err
	}

	ch, err := s.cachedChannelRepo.Get(ctx, channelID)
	if err != nil {
		return err
	}

	if !ch.Type.IsDirect() {
		return ErrOnlyDirectChannelsSupported()
	}

	now := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		_, err := s.repo.UpdateIsVisible(
			txCtx,
			channelID,
			claims.UserID,
			false,
			now,
		)
		if err != nil {
			return err
		}

		payload := EventMemberClosedDirectPayload{
			ChannelID: ch.ID,
			CreatedAt: now,
		}

		event, err := outbox.New(
			claims.UserID,
			claims.SessionID,
			appctx.GetTraceID(txCtx),
			nil,
			EventMemberClosedDirect,
			payload,
			now,
		)
		if err != nil {
			return err
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if err != nil {
		return err
	}

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.cache.Remove(cacheCtx, channelID, claims.UserID); err != nil {
		slog.ErrorContext(cacheCtx, "failed to invalidate member cache on close direct",
			slog.String("channel_id", channelID.String()),
			slog.String("user_id", claims.UserID.String()),
			slog.String("error", err.Error()),
		)
	}

	return nil
}

// UpdateLastReadMessage updates a member's last read message id and timestamp.
func (s *MemberService) UpdateLastReadMessage(
	ctx context.Context,
	channelID uuid.UUID,
	lastReadMessageID *uuid.UUID,
) (*Member, error) {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return nil, err
	}

	ch, err := s.cachedChannelRepo.Get(ctx, channelID)
	if err != nil {
		return nil, err
	}

	var mentionCount *int
	if ch.LastMessageID == lastReadMessageID {
		zero := 0
		mentionCount = &zero
	}

	var updatedMember *Member
	now := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		updatedMember, err = s.repo.UpdateLastReadMessage(
			txCtx,
			channelID,
			claims.UserID,
			lastReadMessageID,
			now,
			now,
			mentionCount,
		)
		if err != nil {
			return err
		}

		payload := EventMemberUpdatedPayload{
			ChannelID:  channelID,
			LastReadID: lastReadMessageID,
			CreatedAt:  now,
		}

		event, err := outbox.New(
			claims.UserID,
			claims.SessionID,
			appctx.GetTraceID(txCtx),
			nil,
			EventMemberUpdated,
			payload,
			now,
		)
		if err != nil {
			return err
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.cache.Remove(cacheCtx, channelID, claims.UserID); err != nil {
		slog.ErrorContext(cacheCtx, "failed to invalidate member cache on update last read message",
			slog.String("channel_id", channelID.String()),
			slog.String("user_id", claims.UserID.String()),
			slog.String("error", err.Error()),
		)
	}

	return updatedMember, nil
}

// UpdatePinnedAt updates a member's pinned at timestamp.
func (s *MemberService) UpdatePinnedAt(
	ctx context.Context,
	channelID uuid.UUID,
	isPinned bool,
) (*Member, error) {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return nil, err
	}

	var updatedMember *Member
	var pinnedAt *time.Time
	now := time.Now()

	if isPinned {
		pinnedAt = &now
	}

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		updatedMember, err = s.repo.UpdatePinnedAt(
			txCtx,
			channelID,
			claims.UserID,
			pinnedAt,
			now,
		)
		if err != nil {
			return err
		}

		payload := EventMemberUpdatedPayload{
			ChannelID: channelID,
			PinnedAt:  pinnedAt,
			CreatedAt: now,
		}

		event, err := outbox.New(
			claims.UserID,
			claims.SessionID,
			appctx.GetTraceID(txCtx),
			nil,
			EventMemberUpdated,
			payload,
			now,
		)
		if err != nil {
			return err
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.cache.Remove(cacheCtx, channelID, claims.UserID); err != nil {
		slog.ErrorContext(cacheCtx, "failed to invalidate member cache on update pinned at",
			slog.String("channel_id", channelID.String()),
			slog.String("user_id", claims.UserID.String()),
			slog.String("error", err.Error()),
		)
	}

	return updatedMember, nil
}

// UpdateMutedUntil updates a member's muted until timestamp.
func (s *MemberService) UpdateMutedUntil(
	ctx context.Context,
	channelID uuid.UUID,
	rawDuration *string,
) (*Member, error) {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return nil, err
	}

	var mutedUntil *time.Time
	now := time.Now()

	if rawDuration != nil {
		muteDuration, err := ParseMuteDurationString(*rawDuration)
		if err != nil {
			return nil, err
		}

		calculated, err := muteDuration.CalculateUntil(now)
		if err != nil {
			return nil, err
		}
		mutedUntil = calculated
	}

	var updatedMember *Member

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		updatedMember, err = s.repo.UpdateMutedUntil(
			txCtx,
			channelID,
			claims.UserID,
			mutedUntil,
			now,
		)
		if err != nil {
			return err
		}

		payload := EventMemberUpdatedPayload{
			ChannelID:  channelID,
			MutedUntil: mutedUntil,
			CreatedAt:  now,
		}

		event, createErr := outbox.New(
			claims.UserID,
			claims.SessionID,
			appctx.GetTraceID(txCtx),
			nil,
			EventMemberUpdated,
			payload,
			now,
		)
		if createErr != nil {
			return createErr
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.cache.Remove(cacheCtx, channelID, claims.UserID); err != nil {
		slog.ErrorContext(cacheCtx, "failed to invalidate member cache on update pinned at",
			slog.String("channel_id", channelID.String()),
			slog.String("user_id", claims.UserID.String()),
			slog.String("error", err.Error()),
		)
	}

	return updatedMember, nil
}

// LeaveGroup deletes a member and a group channel if no remaining members exist.
func (s *MemberService) LeaveGroup(
	ctx context.Context,
	channelID uuid.UUID,
) error {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return err
	}

	var (
		channelDeleted bool
		sysMsg         *Message
		memberIDs      []uuid.UUID
		updatedPeers   []*Member
	)

	now := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		ch, err := s.channelRepo.GetForUpdate(txCtx, channelID)
		if err != nil {
			return err
		}

		if ch.Type.IsDirect() {
			return ErrCannotLeaveDirectChannel()
		}

		existingMembers, err := s.repo.GetBatchByChannelID(txCtx, channelID)
		if err != nil {
			return err
		}

		if err := validateMembership(existingMembers, claims.UserID); err != nil {
			return err
		}

		err = s.repo.Delete(txCtx, channelID, claims.UserID)
		if err != nil {
			return err
		}

		memberIDs = getMemberIDs(existingMembers)
		remainingCount := len(memberIDs) - 1

		if remainingCount <= 0 {
			channelDeleted = true
			if err = s.channelRepo.Delete(txCtx, channelID); err != nil {
				return err
			}

			payload := EventMemberLeftPayload{
				ChannelID: channelID,
				MemberID:  claims.UserID,
				CreatedAt: now,
			}

			event, createErr := outbox.New(
				claims.UserID,
				claims.SessionID,
				appctx.GetTraceID(txCtx),
				memberIDs,
				EventMemberLeft,
				payload,
				now,
			)
			if createErr != nil {
				return createErr
			}

			return s.outboxRepo.Create(txCtx, event)
		}

		msg, err := NewMessageMemberLeave(ch.ID, claims.UserID, now)
		if err != nil {
			return err
		}

		sysMsg, err = s.messageRepo.Create(txCtx, msg)
		if err != nil {
			return err
		}

		updatedPeers, err = s.repo.IncrementPeersMentionCountByChannelID(txCtx, ch.ID, claims.UserID, 1, now)
		if err != nil {
			return err
		}

		// Target only remaining members (exclude the user who left)
		recipientIDs := make([]uuid.UUID, 0, remainingCount)
		for _, id := range memberIDs {
			if id != claims.UserID {
				recipientIDs = append(recipientIDs, id)
			}
		}

		msgView := ParseMessageView(sysMsg)
		membersMap := ParseMemberViewsMap(updatedPeers)

		payload := EventMemberLeftPayload{
			ChannelID:     channelID,
			MemberID:      claims.UserID,
			Members:       &membersMap,
			SystemMessage: &msgView,
			CreatedAt:     now,
		}

		event, createErr := outbox.New(
			claims.UserID,
			claims.SessionID,
			appctx.GetTraceID(txCtx),
			recipientIDs,
			EventMemberLeft,
			payload,
			now,
		)
		if createErr != nil {
			return createErr
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if err != nil {
		return err
	}

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if channelDeleted {
		if err := s.channelCache.Delete(cacheCtx, channelID); err != nil {
			slog.ErrorContext(cacheCtx, "failed to invalidate deleted channel cache on leave group",
				slog.String("channel_id", channelID.String()),
				slog.String("error", err.Error()),
			)
		}
	}

	if err := s.cache.InvalidateChannel(cacheCtx, channelID); err != nil {
		slog.ErrorContext(cacheCtx, "failed to invalidate member cache on leave group",
			slog.String("channel_id", channelID.String()),
			slog.String("error", err.Error()),
		)
	}

	if sysMsg != nil {
		if err := s.messageCache.SetBatch(cacheCtx, channelID, []*Message{sysMsg}); err != nil {
			slog.ErrorContext(cacheCtx, "failed to seed system leave message into cache",
				slog.String("channel_id", channelID.String()),
				slog.String("error", err.Error()),
			)
		}
	}

	return nil
}

func filterNewMemberIDs(existingMembers []*Member, newPeerIDs []uuid.UUID, actorID uuid.UUID) ([]uuid.UUID, error) {
	existingSet := make(map[uuid.UUID]struct{}, len(existingMembers))
	for _, m := range existingMembers {
		existingSet[m.UserID] = struct{}{}
	}

	toAddIDs := make([]uuid.UUID, 0, len(newPeerIDs))
	for _, id := range newPeerIDs {
		if id == actorID {
			continue
		}
		if _, exists := existingSet[id]; !exists {
			toAddIDs = append(toAddIDs, id)
		}
	}

	if len(toAddIDs) == 0 {
		return nil, ErrAlreadyMembers()
	}

	if len(existingMembers)+len(toAddIDs) > ChannelMaxPeers+1 {
		return nil, ErrMaxCapacityExceeded()
	}

	return toAddIDs, nil
}
