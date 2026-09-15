package channel

import (
	"bonfire-api/internal/helpers"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type ChannelService struct {
	cache          ChannelCache
	repo           ChannelRepository
	memberRepo     MemberRepository
	messageRepo    MessageRepository
	presenceCache  PresenceCache
	cachedUserRepo CachedUserRepository
	outboxRepo     OutboxRepository
	relationRepo   RelationRepository
	tx             TX
}

func NewChannelService(
	cache ChannelCache,
	repo ChannelRepository,
	memberRepo MemberRepository,
	messageRepo MessageRepository,
	reactionRepo ReactionRepository,
	presenceCache PresenceCache,
	cachedUserRepo CachedUserRepository,
	outboxRepo OutboxRepository,
	relationRepo RelationRepository,
	tx TX,
) *ChannelService {
	return &ChannelService{
		cache:          cache,
		repo:           repo,
		memberRepo:     memberRepo,
		messageRepo:    messageRepo,
		presenceCache:  presenceCache,
		cachedUserRepo: cachedUserRepo,
		outboxRepo:     outboxRepo,
		relationRepo:   relationRepo,
		tx:             tx,
	}
}

type CreateGroupResult struct {
	Channel     *Channel
	ActorMember *Member
	Users       map[uuid.UUID]*user.User
	Presences   map[uuid.UUID]presence.Presence
	MemberIDs   []uuid.UUID
}

// CreateGroup creates a new group channel with members.
func (s *ChannelService) CreateGroup(ctx context.Context, actorID, sessionID uuid.UUID, rawPeerIDs []uuid.UUID) (*CreateGroupResult, error) {
	if err := validateMaxPeers(rawPeerIDs); err != nil {
		return nil, err
	}

	dedupedMemberIDs := helpers.DedupeIDs(append(rawPeerIDs, actorID))
	peerIDs := helpers.RemoveID(actorID, dedupedMemberIDs)

	if len(peerIDs) > 0 {
		if err := s.relationRepo.HasIncomingBlock(ctx, actorID, peerIDs); err != nil {
			return nil, err
		}
	}

	now := time.Now()

	ch, err := NewGroupChannel(now)
	if err != nil {
		return nil, err
	}

	membs := NewMembers(ch.ID, actorID, peerIDs, now)

	g, gCtx := errgroup.WithContext(ctx)

	var (
		users     map[uuid.UUID]*user.User
		presences map[uuid.UUID]presence.Presence
	)

	g.Go(func() error {
		var fetchErr error
		users, fetchErr = s.cachedUserRepo.GetBatchValid(gCtx, dedupedMemberIDs)
		return fetchErr
	})

	g.Go(func() error {
		var fetchErr error
		presences, fetchErr = s.presenceCache.GetBatchPresence(gCtx, dedupedMemberIDs)
		return fetchErr
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	sortMemberIDs(dedupedMemberIDs, users)

	txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var repoErr error
		if ch, repoErr = s.repo.Create(txCtx, ch); repoErr != nil {
			return repoErr
		}

		if membs, repoErr = s.memberRepo.CreateBatch(txCtx, membs); repoErr != nil {
			return repoErr
		}

		membersMap := make(map[uuid.UUID]*Member, len(membs))
		for _, m := range membs {
			membersMap[m.UserID] = m
		}

		payload := EventChannelCreatedPayload{
			ExcludeSessionID: sessionID,
			Channel:          ch,
			Users:            users,
			Presences:        presences,
			MemberIDs:        dedupedMemberIDs,
		}

		return s.outboxRepo.Publish(txCtx, EventChannelCreated, payload, now)
	})
	if txErr != nil {
		return nil, txErr
	}

	_ = s.cache.CreateGroup(ctx, ch, membs)

	return &CreateGroupResult{
		Channel:     ch,
		ActorMember: filterMembership(actorID, membs),
		Users:       users,
		Presences:   presences,
		MemberIDs:   dedupedMemberIDs,
	}, nil
}

// UpdateGroup updates the group channel properties name and icon_url.
func (s *ChannelService) UpdateGroup(ctx context.Context, actorID, sessionID, channelID uuid.UUID, name, iconURL *string) (*Channel, error) {
	members, err := s.memberRepo.GetBatchByChannelID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	_, err = validateMembership(actorID, members)
	if err != nil {
		return nil, err
	}

	var channel *Channel
	now := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		channel, err = s.repo.UpdateGroup(txCtx, channelID, name, iconURL, now)
		if err != nil {
			return err
		}

		systemMessages, err := buildUpdateGroupSystemMessages(channel.ID, ptr.To(actorID), name, iconURL, now)
		if err != nil {
			return err
		}

		if len(systemMessages) > 0 {
			if _, err := s.messageRepo.CreateBatchAndMention(
				txCtx,
				systemMessages,
				channel.ID,
				actorID,
				now,
			); err != nil {
				return err
			}
		}

		payload := EventChannelUpdatedPayload{
			ExcludeSessionID: sessionID,
			Channel:          channel,
			MemberIDs:        getMemberIDs(members),
		}

		return s.outboxRepo.Publish(txCtx, EventChannelUpdated, payload, now)
	})
	if err != nil {
		return nil, err
	}

	_ = s.cache.Delete(ctx, channel.ID)

	return channel, nil
}

func buildUpdateGroupSystemMessages(
	channelID uuid.UUID,
	actorID *uuid.UUID,
	name *string,
	iconURL *string,
	now time.Time,
) ([]*Message, error) {
	var systemMessages []*Message

	if name != nil {
		msg, err := NewMessageNameChange(channelID, actorID, name, now)
		if err != nil {
			return nil, err
		}
		systemMessages = append(systemMessages, msg)
	}

	if iconURL != nil {
		iconTime := now
		if len(systemMessages) > 0 {
			iconTime = now.Add(time.Microsecond)
		}

		msg, err := NewMessageIconChange(channelID, actorID, iconTime)
		if err != nil {
			return nil, err
		}
		systemMessages = append(systemMessages, msg)
	}

	return systemMessages, nil
}
