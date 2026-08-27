package channel

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/pkg/ptr"
	"context"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type MemberService struct {
	repo         MemberRepository
	channelRepo  ChannelRepository
	messageRepo  MessageRepository
	userRepo     UserRepository
	userCache    UserCache
	outboxRepo   OutboxRepository
	relationRepo RelationRepository
	tx           TX
}

func NewMemberService(
	repo MemberRepository,
	channelRepo ChannelRepository,
	messageRepo MessageRepository,
	userRepo UserRepository,
	userCache UserCache,
	outboxRepo OutboxRepository,
	relationRepo RelationRepository,
	tx TX,
) *MemberService {
	return &MemberService{
		repo:         repo,
		channelRepo:  channelRepo,
		messageRepo:  messageRepo,
		userRepo:     userRepo,
		userCache:    userCache,
		outboxRepo:   outboxRepo,
		relationRepo: relationRepo,
		tx:           tx,
	}
}

// AddMembers adds members to a channel and creates a group channel if adding to a direct channel.
func (s *MemberService) AddMembers(
	ctx context.Context,
	rawActorID,
	rawChannelID uuid.UUID,
	rawMemberIDs []uuid.UUID,
) error {
	if err := validateMinMembers(rawMemberIDs); err != nil {
		return err
	}

	actorID, channelID, err := validateIDs(rawActorID, rawChannelID)
	if err != nil {
		return err
	}

	memberIDs, err := fields.ParseIDs(rawMemberIDs)
	if err != nil {
		return err
	}

	newPeerIDs, err := filterRequiredPeerIDs(actorID, memberIDs)
	if err != nil {
		return err
	}

	var existingMembers []*Member

	g, ctxGrp := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		err = s.relationRepo.HasIncomingBlock(ctxGrp, actorID, newPeerIDs)
		return err
	})

	g.Go(func() error {
		var err error
		existingMembers, err = s.repo.GetBatchByChannelID(ctxGrp, channelID)
		return err
	})

	if err := g.Wait(); err != nil {
		return err
	}

	if _, err := validateMembership(actorID, existingMembers); err != nil {
		return err
	}

	newMemberIDs, err := filterNewMemberIDs(actorID, existingMembers, newPeerIDs)
	if err != nil {
		return err
	}

	now := fields.Now()

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		chLock, err := s.channelRepo.GetForUpdate(txCtx, channelID)
		if err != nil {
			return err
		}

		targetChannelID := channelID
		if chLock.chType.IsDirect() {
			newGroupChannel, err := NewGroupChannel(now)
			if err != nil {
				return err
			}

			createdChannel, err := s.channelRepo.Create(txCtx, newGroupChannel)
			if err != nil {
				return err
			}
			targetChannelID = createdChannel.ID()
		}

		membersToInsert, systemMessages, err := buildAddMemberPayloads(
			chLock.chType.IsDirect(), targetChannelID, actorID, newMemberIDs, now,
		)
		if err != nil {
			return err
		}

		if len(systemMessages) > 0 {
			if _, err := s.messageRepo.CreateBatchAndMention(
				txCtx,
				systemMessages,
				targetChannelID,
				actorID,
				now,
			); err != nil {
				return err
			}
		}

		if _, err := s.repo.CreateBatch(txCtx, membersToInsert); err != nil {
			return err
		}

		return s.outboxRepo.Publish(txCtx, EventMembersAdded, MembersAddedPayload{})
	})
}

func buildAddMembersSystemMessages(
	channelID, actorID fields.ID,
	newMemberIDs []fields.ID,
	now fields.Timestamp,
) ([]*Message, error) {
	if len(newMemberIDs) == 0 {
		return nil, nil
	}

	systemMessages := make([]*Message, 0, len(newMemberIDs))
	msgTime := now

	for _, addedUserID := range newMemberIDs {
		msg, err := NewMessageMemberAdd(channelID, actorID, addedUserID, msgTime)
		if err != nil {
			return nil, err
		}
		systemMessages = append(systemMessages, msg)
		msgTime = msgTime.Add(time.Microsecond)
	}

	return systemMessages, nil
}

func buildAddMemberPayloads(
	isDirect bool,
	channelID, actorID fields.ID,
	newMemberIDs []fields.ID,
	now fields.Timestamp,
) (members []*Member, systemMessages []*Message, err error) {
	if isDirect {
		return NewMembers(channelID, actorID, newMemberIDs, now), nil, nil
	}

	systemMessages, err = buildAddMembersSystemMessages(channelID, actorID, newMemberIDs, now)
	if err != nil {
		return nil, nil, err
	}

	return NewPeers(channelID, newMemberIDs, now), systemMessages, nil
}

// CloseDirect updates the visibility of a channel membership to false.
func (s *MemberService) CloseDirect(
	ctx context.Context,
	rawActorID,
	rawChannelID uuid.UUID,
) error {
	actorID, channelID, err := validateIDs(rawActorID, rawChannelID)
	if err != nil {
		return err
	}

	ch, err := s.channelRepo.Get(ctx, channelID)
	if err != nil {
		return err
	}

	if !ch.Type().IsDirect() {
		return ErrOnlyDirectChannelsSupported()
	}

	now := fields.Now()

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		_, err := s.repo.UpdateIsVisible(
			txCtx,
			channelID,
			actorID,
			false,
			now,
		)
		if err != nil {
			return err
		}

		return s.outboxRepo.Publish(
			txCtx,
			EventMemberUpdateUpdateVisibility,
			MemberUpdateVisibilityPayload{},
		)
	})
}

// UpdateLastReadMessage updates a members last read message id and time.
func (s *MemberService) UpdateLastReadMessage(
	ctx context.Context,
	rawActorID,
	rawChannelID,
	rawLastReadMessageID uuid.UUID,
) (*Member, error) {
	actorID, channelID, lastReadMessageID, err := validateMessageIDs(rawActorID, rawChannelID, rawLastReadMessageID)
	if err != nil {
		return nil, err
	}

	ch, err := s.channelRepo.Get(ctx, channelID)
	if err != nil {
		return nil, err
	}

	var mentionCount *int

	if ch.LastMessageID().Equals(lastReadMessageID) {
		zero := 0
		mentionCount = &zero
	}

	var updatedMember *Member

	now := fields.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		updatedMember, err = s.repo.UpdateLastReadMessage(
			txCtx,
			channelID,
			actorID,
			lastReadMessageID,
			now,
			now,
			mentionCount,
		)
		if err != nil {
			return err
		}

		return s.outboxRepo.Publish(
			txCtx,
			EventMemberUpdateLastReadMessage,
			MemberUpdateLastReadMessagePayload{},
		)
	})
	if err != nil {
		return nil, err
	}

	return updatedMember, nil
}

// UpdatePinnedAt updates a members pinned at timestamp.
func (s *MemberService) UpdatePinnedAt(
	ctx context.Context,
	rawActorID,
	rawChannelID uuid.UUID,
	isPinned bool,
) (*Member, error) {
	actorID, channelID, err := validateIDs(rawActorID, rawChannelID)
	if err != nil {
		return nil, err
	}

	var updatedMember *Member
	pinnedAt := fields.Timestamp{}
	now := fields.Now()

	if isPinned {
		pinnedAt = now
	}

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		updatedMember, err = s.repo.UpdatePinnedAt(
			txCtx,
			channelID,
			actorID,
			pinnedAt,
			now,
		)
		if err != nil {
			return err
		}

		return s.outboxRepo.Publish(
			txCtx,
			EventMemberUpdatePinnedAt,
			MemberUpdatePinnedAtPayload{},
		)
	})
	if err != nil {
		return nil, err
	}

	return updatedMember, nil
}

// UpdateMutedUntil updates a members muted until timestamp.
func (s *MemberService) UpdateMutedUntil(
	ctx context.Context,
	rawActorID,
	rawChannelID uuid.UUID,
	rawDuration *int,
) (*Member, error) {
	actorID, channelID, err := validateIDs(rawActorID, rawChannelID)
	if err != nil {
		return nil, err
	}

	mutedUntil := fields.Timestamp{}
	now := fields.Now()

	if rawDuration != nil {
		muteDuration, err := ParseMuteDuration(ptr.From(rawDuration))
		if err != nil {
			return nil, err
		}

		mutedUntil, err = muteDuration.CalculateUntil(now)
		if err != nil {
			return nil, err
		}
	}

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

		return s.outboxRepo.Publish(
			txCtx,
			EventMemberUpdateMutedUntil,
			MemberUpdateMutedUntilPayload{},
		)
	})
	if err != nil {
		return nil, err
	}

	return updatedMember, nil
}

// LeaveGroup deletes a member and a group channel if no remaining members.
func (s *MemberService) LeaveGroup(
	ctx context.Context,
	rawActorID,
	rawChannelID uuid.UUID,
) error {
	actorID, channelID, err := validateIDs(rawActorID, rawChannelID)
	if err != nil {
		return err
	}

	now := fields.Now()

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		ch, err := s.channelRepo.GetForUpdate(txCtx, channelID)
		if err != nil {
			return err
		}

		if ch.Type().IsDirect() {
			return ErrCannotLeaveDirectChannel()
		}

		err = s.repo.Delete(txCtx, channelID, actorID)
		if err != nil {
			return err
		}

		remainingCount, err := s.repo.CountByChannelID(txCtx, channelID)
		if err != nil {
			return err
		}

		if remainingCount == 0 {
			err = s.channelRepo.Delete(txCtx, channelID)
			if err != nil {
				return err
			}
			return nil
		}

		sysMsg, err := NewMessageMemberLeave(ch.id, actorID, now)

		_, err = s.messageRepo.CreateAndMention(txCtx, sysMsg, ch.ID(), actorID, now)
		if err != nil {
			return err
		}

		return s.outboxRepo.Publish(
			txCtx,
			EventMemberDelete,
			MemberDeletePayload{},
		)
	})
}

func filterNewMemberIDs(actorID fields.ID, existingMembers []*Member, newPeerIDs []fields.ID) ([]fields.ID, error) {
	existingSet := make(map[fields.ID]struct{}, len(existingMembers))
	for _, m := range existingMembers {
		existingSet[m.UserID()] = struct{}{}
	}

	toAddIDs := make([]fields.ID, 0, len(newPeerIDs))
	for _, id := range newPeerIDs {
		if id.Equals(actorID) {
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
