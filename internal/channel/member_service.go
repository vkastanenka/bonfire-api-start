package channel

import (
	"bonfire-api/internal/helpers"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type MemberService struct {
	repo              MemberRepository
	channelCache      ChannelCache
	channelRepo       ChannelRepository
	cachedChannelRepo CachedChannelRepository
	messageRepo       MessageRepository
	userCache         UserCache
	userRepo          UserRepository
	cachedUserRepo    CachedUserRepository
	presenceCache     PresenceCache
	outboxRepo        OutboxRepository
	relationRepo      RelationRepository
	tx                TX
}

func NewMemberService(
	repo MemberRepository,
	channelCache ChannelCache,
	channelRepo ChannelRepository,
	cachedChannelRepo CachedChannelRepository,
	messageRepo MessageRepository,
	userCache UserCache,
	userRepo UserRepository,
	cachedUserRepo CachedUserRepository,
	presenceCache PresenceCache,
	outboxRepo OutboxRepository,
	relationRepo RelationRepository,
	tx TX,
) *MemberService {
	return &MemberService{
		repo:              repo,
		channelCache:      channelCache,
		channelRepo:       channelRepo,
		cachedChannelRepo: cachedChannelRepo,
		messageRepo:       messageRepo,
		userCache:         userCache,
		userRepo:          userRepo,
		cachedUserRepo:    cachedUserRepo,
		presenceCache:     presenceCache,
		outboxRepo:        outboxRepo,
		relationRepo:      relationRepo,
		tx:                tx,
	}
}

// GetBatchByChannelIDs retrieves members for multiple channels using a cache-aside strategy.
func (s *MemberService) GetBatchByChannelIDs(
	ctx context.Context,
	channelIDs []uuid.UUID,
) (map[uuid.UUID][]*Member, error) {
	if len(channelIDs) == 0 {
		return make(map[uuid.UUID][]*Member), nil
	}

	channelIDs = helpers.DedupeIDs(channelIDs)

	// 1. Attempt cache lookup
	found, missing, err := s.channelCache.GetBatchMembersByChannelIDs(ctx, channelIDs)
	if err != nil {
		// Log cache error if needed; proceed or return error depending on degradation strategy
		return nil, err
	}

	// 2. Return early if all requested channel memberships were cached
	if len(missing) == 0 {
		return found, nil
	}

	// 3. Fetch missing channel memberships from repository
	dbMembersMap, err := s.repo.GetBatchByChannelIDs(ctx, missing)
	if err != nil {
		return nil, err
	}

	if len(dbMembersMap) == 0 {
		return found, nil
	}

	// 4. Backfill cache asynchronously or inline for missing hits
	_ = s.channelCache.SetBatchMembers(ctx, dbMembersMap)

	// 5. Merge DB results into result map
	for cid, members := range dbMembersMap {
		found[cid] = members
	}

	return found, nil
}

// GetBatchByChannelID convenience wrapper for single channel lookups.
func (s *MemberService) GetBatchByChannelID(
	ctx context.Context,
	channelID uuid.UUID,
) ([]*Member, error) {
	res, err := s.GetBatchByChannelIDs(ctx, []uuid.UUID{channelID})
	if err != nil {
		return nil, err
	}
	return res[channelID], nil
}

type AddMembersResult struct {
	Users     map[uuid.UUID]*user.User
	Presences map[uuid.UUID]presence.Presence
	MemberIDs []uuid.UUID
	Messages  []*Message
}

// AddMembers adds members to a channel and creates system notification messages.
func (s *MemberService) AddMembers(
	ctx context.Context,
	actorID, sessionID, channelID uuid.UUID,
	memberIDs []uuid.UUID,
) (*AddMembersResult, error) {
	if err := validateMinMembers(memberIDs); err != nil {
		return nil, err
	}

	newPeerIDs, err := filterRequiredPeerIDs(actorID, memberIDs)
	if err != nil {
		return nil, err
	}

	var (
		existingMembers []*Member
		ch              *Channel
	)

	g, ctxGrp := errgroup.WithContext(ctx)

	g.Go(func() error {
		return s.relationRepo.HasIncomingBlock(ctxGrp, actorID, newPeerIDs)
	})

	g.Go(func() error {
		var fetchErr error
		existingMembers, fetchErr = s.GetBatchByChannelID(ctxGrp, channelID)
		return fetchErr
	})

	g.Go(func() error {
		var fetchErr error
		ch, fetchErr = s.cachedChannelRepo.Get(ctxGrp, channelID)
		return fetchErr
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	if ch.Type.IsDirect() {
		return nil, errors.New("Cannot add members to direct channel.")
	}

	if _, err := validateMembership(actorID, existingMembers); err != nil {
		return nil, err
	}

	newMemberIDs, err := filterNewMemberIDs(actorID, existingMembers, newPeerIDs)
	if err != nil {
		return nil, err
	}

	existingMemberIDs := getMemberIDs(existingMembers)
	allMemberIDs := helpers.DedupeIDs(append(existingMemberIDs, newMemberIDs...))

	var (
		allUsers     map[uuid.UUID]*user.User
		allPresences map[uuid.UUID]presence.Presence
	)

	gHydrate, ctxHydrate := errgroup.WithContext(ctx)

	gHydrate.Go(func() error {
		var fetchErr error
		allUsers, fetchErr = s.cachedUserRepo.GetBatchValid(ctxHydrate, allMemberIDs)
		return fetchErr
	})

	gHydrate.Go(func() error {
		var fetchErr error
		allPresences, fetchErr = s.presenceCache.GetBatchPresence(ctxHydrate, allMemberIDs)
		return fetchErr
	})

	if err := gHydrate.Wait(); err != nil {
		return nil, err
	}

	sortMemberIDs(allMemberIDs, allUsers)

	addedUsers := make(map[uuid.UUID]*user.User, len(newMemberIDs))
	addedPresences := make(map[uuid.UUID]presence.Presence, len(newMemberIDs))
	for _, id := range newMemberIDs {
		if u, ok := allUsers[id]; ok {
			addedUsers[id] = u
		}
		if p, ok := allPresences[id]; ok {
			addedPresences[id] = p
		}
	}

	now := time.Now()
	var (
		createdMessages []*Message
		membersToInsert []*Member
	)

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		chLock, err := s.channelRepo.GetForUpdate(txCtx, channelID)
		if err != nil {
			return err
		}

		if chLock.Type.IsDirect() {
			return errors.New("Cannot add members to direct channel.")
		}

		systemMessages, err := buildAddMembersSystemMessages(chLock.ID, &actorID, newMemberIDs, now)
		if err != nil {
			return err
		}

		createdMessages, err = s.messageRepo.CreateBatchAndMention(
			txCtx,
			systemMessages,
			chLock.ID,
			actorID,
			now,
		)
		if err != nil {
			return err
		}

		membersToInsert = NewPeers(chLock.ID, newMemberIDs, now)
		if _, err := s.repo.CreateBatch(txCtx, membersToInsert); err != nil {
			return err
		}

		sortMessages(createdMessages)

		payload := EventMembersAddedPayload{
			ExcludeSessionID: sessionID,
			Channel:          chLock,
			Users:            allUsers,
			Presences:        allPresences,
			MemberIDs:        allMemberIDs,
			SystemMessages:   createdMessages,
		}

		return s.outboxRepo.Publish(txCtx, EventMembersAdded, payload, now)
	})
	if err != nil {
		return nil, err
	}

	_ = s.channelCache.AddMembers(ctx, channelID, membersToInsert)

	return &AddMembersResult{
		Users:     addedUsers,
		Presences: addedPresences,
		MemberIDs: allMemberIDs,
		Messages:  createdMessages,
	}, nil
}

func buildAddMembersSystemMessages(
	channelID uuid.UUID,
	actorID *uuid.UUID,
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
func (s *MemberService) CloseDirect(
	ctx context.Context,
	actorID,
	sessionID,
	channelID uuid.UUID,
) error {
	ch, err := s.channelRepo.Get(ctx, channelID)
	if err != nil {
		return err
	}

	if !ch.Type.IsDirect() {
		return ErrOnlyDirectChannelsSupported()
	}

	now := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		member, err := s.repo.UpdateIsVisible(
			txCtx,
			channelID,
			actorID,
			false,
			now,
		)
		if err != nil {
			return err
		}

		payload := EventMemberClosedDirectPayload{
			ExcludeSessionID: sessionID,
			MemberID:         member.UserID,
			ChannelID:        ch.ID,
		}

		return s.outboxRepo.Publish(
			txCtx,
			EventMemberClosedDirect,
			payload,
			now,
		)
	})
	if err != nil {
		return err
	}

	_ = s.userCache.RemoveChannelID(ctx, actorID, channelID)

	return nil
}

// UpdateLastReadMessage updates a member's last read message id and timestamp.
func (s *MemberService) UpdateLastReadMessage(
	ctx context.Context,
	actorID,
	sessionID,
	channelID,
	lastReadMessageID uuid.UUID,
) (*Member, error) {
	ch, err := s.channelRepo.Get(ctx, channelID)
	if err != nil {
		return nil, err
	}

	var mentionCount *int
	if ch.LastMessageID == &lastReadMessageID {
		zero := 0
		mentionCount = &zero
	}

	var updatedMember *Member
	now := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		updatedMember, err = s.repo.UpdateLastReadMessage(
			txCtx,
			channelID,
			actorID,
			&lastReadMessageID,
			now,
			now,
			mentionCount,
		)
		if err != nil {
			return err
		}

		payload := EventMemberUpdatedPayload{
			ExcludeSessionID: sessionID,
			ChannelID:        channelID,
			MemberID:         actorID,
			LastReadID:       &lastReadMessageID,
		}

		return s.outboxRepo.Publish(
			txCtx,
			EventMemberUpdated,
			payload,
			now,
		)
	})
	if err != nil {
		return nil, err
	}

	_ = s.channelCache.InvalidateMember(ctx, channelID, actorID)

	return updatedMember, nil
}

// UpdatePinnedAt updates a member's pinned at timestamp.
func (s *MemberService) UpdatePinnedAt(
	ctx context.Context,
	actorID,
	sessionID,
	channelID uuid.UUID,
	isPinned bool,
) (*Member, error) {
	var err error
	var updatedMember *Member
	pinnedAt := time.Time{}
	now := time.Now()

	if isPinned {
		pinnedAt = now
	}

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		updatedMember, err = s.repo.UpdatePinnedAt(
			txCtx,
			channelID,
			actorID,
			&pinnedAt,
			now,
		)
		if err != nil {
			return err
		}

		payload := EventMemberUpdatedPayload{
			ExcludeSessionID: sessionID,
			ChannelID:        channelID,
			MemberID:         actorID,
			PinnedAt:         &pinnedAt,
		}

		return s.outboxRepo.Publish(
			txCtx,
			EventMemberUpdated,
			payload,
			now,
		)
	})
	if err != nil {
		return nil, err
	}

	_ = s.channelCache.InvalidateMember(ctx, channelID, actorID)

	return updatedMember, nil
}

// UpdateMutedUntil updates a member's muted until timestamp.
func (s *MemberService) UpdateMutedUntil(
	ctx context.Context,
	actorID,
	sessionID,
	channelID uuid.UUID,
	rawDuration *int,
) (*Member, error) {
	var mutedUntil *time.Time
	now := time.Now()

	if rawDuration != nil {
		muteDuration, err := ParseMuteDuration(*rawDuration)
		if err != nil {
			return nil, err
		}

		calculated, err := muteDuration.CalculateUntil(now)
		if err != nil {
			return nil, err
		}
		mutedUntil = calculated
	}

	var err error
	var updatedMember *Member

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		updatedMember, err = s.repo.UpdateMutedUntil(
			txCtx,
			channelID,
			actorID,
			mutedUntil,
			now,
		)
		if err != nil {
			return err
		}

		payload := EventMemberUpdatedPayload{
			ExcludeSessionID: sessionID,
			ChannelID:        channelID,
			MemberID:         actorID,
			MutedUntil:       mutedUntil,
		}

		return s.outboxRepo.Publish(
			txCtx,
			EventMemberUpdated,
			payload,
			now,
		)
	})
	if err != nil {
		return nil, err
	}

	_ = s.channelCache.InvalidateMember(ctx, channelID, actorID)

	return updatedMember, nil
}

// LeaveGroup deletes a member and a group channel if no remaining members exist.
func (s *MemberService) LeaveGroup(
	ctx context.Context,
	actorID,
	sessionID,
	channelID uuid.UUID,
) error {
	now := time.Now()
	var (
		channelDeleted bool
		sysMsg         *Message
		memberIDs      []uuid.UUID
	)

	err := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
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

		memberIDs = make([]uuid.UUID, len(existingMembers))
		for i, m := range existingMembers {
			memberIDs[i] = m.UserID
		}

		err = s.repo.Delete(txCtx, channelID, actorID)
		if err != nil {
			return err
		}

		remainingCount := len(existingMembers) - 1

		if remainingCount <= 0 {
			channelDeleted = true
			if err = s.channelRepo.Delete(txCtx, channelID); err != nil {
				return err
			}

			payload := EventMemberLeftPayload{
				ExcludeSessionID: sessionID,
				ActorID:          actorID,
				ChannelID:        channelID,
				MemberIDs:        memberIDs,
				SystemMessage:    nil,
			}

			return s.outboxRepo.Publish(
				txCtx,
				EventMemberLeft,
				payload,
				now,
			)
		}

		msg, err := NewMessageMemberLeave(ch.ID, &actorID, now)
		if err != nil {
			return err
		}

		sysMsg, err = s.messageRepo.CreateAndMention(txCtx, msg, ch.ID, actorID, now)
		if err != nil {
			return err
		}

		payload := EventMemberLeftPayload{
			ExcludeSessionID: sessionID,
			ActorID:          actorID,
			ChannelID:        channelID,
			MemberIDs:        memberIDs,
			SystemMessage:    sysMsg,
		}

		return s.outboxRepo.Publish(
			txCtx,
			EventMemberLeft,
			payload,
			now,
		)
	})
	if err != nil {
		return err
	}

	if channelDeleted {
		_ = s.channelCache.InvalidateMembers(ctx, channelID)
	} else {
		_ = s.channelCache.InvalidateMember(ctx, channelID, actorID)
	}

	_ = s.userCache.RemoveChannelID(ctx, actorID, channelID)

	return nil
}

func filterNewMemberIDs(actorID uuid.UUID, existingMembers []*Member, newPeerIDs []uuid.UUID) ([]uuid.UUID, error) {
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
