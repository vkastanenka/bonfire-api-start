package channel

import (
	"bonfire-api/internal/appctx"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type ChannelService struct {
	cache              ChannelCache
	repo               ChannelRepository
	memberCache        MemberCache
	memberRepo         MemberRepository
	messageCache       MessageCache
	messageRepo        MessageRepository
	presenceCache      PresenceCache
	cachedUserRepo     CachedUserRepository
	outboxRepo         OutboxRepository
	relationRepo       RelationRepository
	cachedRelationRepo CachedRelationRepository
	tx                 TX
}

func NewChannelService(
	cache ChannelCache,
	repo ChannelRepository,
	memberCache MemberCache,
	memberRepo MemberRepository,
	messageCache MessageCache,
	messageRepo MessageRepository,
	presenceCache PresenceCache,
	cachedUserRepo CachedUserRepository,
	outboxRepo OutboxRepository,
	relationRepo RelationRepository,
	cachedRelationRepo CachedRelationRepository,
	tx TX,
) *ChannelService {
	return &ChannelService{
		cache:              cache,
		repo:               repo,
		memberCache:        memberCache,
		memberRepo:         memberRepo,
		messageCache:       messageCache,
		messageRepo:        messageRepo,
		presenceCache:      presenceCache,
		cachedUserRepo:     cachedUserRepo,
		outboxRepo:         outboxRepo,
		relationRepo:       relationRepo,
		cachedRelationRepo: cachedRelationRepo,
		tx:                 tx,
	}
}

type CreateGroupResult struct {
	Channel     *Channel
	MemberIDs   []uuid.UUID
	ActorMember *Member
	Users       map[uuid.UUID]*user.User
	Presences   map[uuid.UUID]presence.Presence
}

// CreateGroup creates a new group channel with members and returns the initialized result for the actor.
func (s *ChannelService) CreateGroup(ctx context.Context, rawMemberIDs []uuid.UUID) (*CreateGroupResult, error) {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return nil, err
	}

	if err := validateMaxPeers(rawMemberIDs); err != nil {
		return nil, err
	}

	memberIDs, peerIDs := getChMemberIDs(rawMemberIDs, claims.UserID)

	counts, err := s.memberRepo.CountBatchByUserID(ctx, memberIDs)
	if err != nil {
		return nil, err
	}

	if counts[claims.UserID] >= MaxUserMemberships {
		return nil, ErrUserMaxChannelsReached()
	}

	if len(peerIDs) > 0 {
		for _, peerID := range peerIDs {
			if counts[peerID] >= MaxUserMemberships {
				return nil, ErrPeerMaxChannelsReached()
			}
		}

		friendIDs, err := s.cachedRelationRepo.GetFriendIDs(ctx, claims.UserID)
		if err != nil {
			return nil, err
		}

		friendSet := make(map[uuid.UUID]struct{}, len(friendIDs))
		for _, id := range friendIDs {
			friendSet[id] = struct{}{}
		}

		for _, peerID := range peerIDs {
			if _, isFriend := friendSet[peerID]; !isFriend {
				return nil, ErrCannotAddNonFriendUserToGroup()
			}
		}
	}

	now := time.Now()

	ch, err := NewGroupChannel(now)
	if err != nil {
		return nil, err
	}

	membs := NewMembers(ch.ID, claims.UserID, peerIDs, now)

	g, gCtx := errgroup.WithContext(ctx)

	var (
		users     map[uuid.UUID]*user.User
		presences map[uuid.UUID]presence.Presence
	)

	g.Go(func() error {
		var fetchErr error
		users, fetchErr = s.cachedUserRepo.GetBatch(gCtx, memberIDs)
		return fetchErr
	})

	g.Go(func() error {
		var fetchErr error
		presences, fetchErr = s.presenceCache.GetBatch(gCtx, memberIDs)
		return fetchErr
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var actorMember *Member

	txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var repoErr error
		if ch, repoErr = s.repo.Create(txCtx, ch); repoErr != nil {
			return repoErr
		}

		if membs, repoErr = s.memberRepo.CreateBatch(txCtx, membs); repoErr != nil {
			return repoErr
		}

		for _, m := range membs {
			if m.UserID == claims.UserID {
				actorMember = m
				break
			}
		}

		payload := EventChannelCreatedPayload{
			Channel:   ParseChannelView(ch),
			Members:   ParseMemberViewsMap(membs),
			Users:     user.ParseSummariesMap(users),
			Presences: presences,
			CreatedAt: now,
		}

		event, err := outbox.New(
			claims.UserID,
			claims.SessionID,
			appctx.GetTraceID(ctx),
			memberIDs,
			EventChannelCreated,
			payload,
			now,
		)
		if err != nil {
			return err
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if txErr != nil {
		return nil, txErr
	}

	sortMembers(membs, users)
	memberIDs = getMemberIDs(membs)

	cacheCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := s.cache.CreateGroup(cacheCtx, ch, membs); err != nil {
		slog.ErrorContext(cacheCtx, "failed to seed created group in cache",
			slog.String("channel_id", ch.ID.String()),
			slog.Any("error", err),
		)
	}

	return &CreateGroupResult{
		Channel:     ch,
		MemberIDs:   memberIDs,
		ActorMember: actorMember,
		Users:       users,
		Presences:   presences,
	}, nil
}

type UpdateGroupResult struct {
	Channel        *Channel
	SystemMessages []*Message
}

// UpdateGroup updates the group channel properties name and icon_url.
func (s *ChannelService) UpdateGroup(ctx context.Context, channelID uuid.UUID, name, iconURL *string) (*UpdateGroupResult, error) {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return nil, err
	}

	members, err := s.memberRepo.GetBatchByChannelID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	err = validateMembership(members, claims.UserID)
	if err != nil {
		return nil, err
	}

	memberIDs := getMemberIDs(members)
	var updatedChannel *Channel
	var systemMessages []*Message
	var updatedPeers []*Member
	now := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		updatedChannel, err = s.repo.UpdateGroup(txCtx, channelID, name, iconURL, now)
		if err != nil {
			return err
		}

		systemMessages, err = buildUpdateGroupSystemMessages(updatedChannel.ID, claims.UserID, name, iconURL, now)
		if err != nil {
			return err
		}

		systemMessagesLength := len(systemMessages)

		if systemMessagesLength > 0 {
			if systemMessages, err = s.messageRepo.CreateBatch(txCtx, systemMessages); err != nil {
				return err
			}

			updatedPeers, err = s.memberRepo.IncrementPeersMentionCountByChannelID(txCtx, updatedChannel.ID, claims.UserID, systemMessagesLength, now)
			if err != nil {
				return err
			}
		}

		sortMessages(systemMessages)

		payload := EventChannelUpdatedPayload{
			Channel:        ParseChannelView(updatedChannel),
			Members:        ParseMemberViewsMap(updatedPeers),
			SystemMessages: ParseMessageViews(systemMessages),
			CreatedAt:      now,
		}

		event, repoErr := outbox.New(
			claims.UserID,
			claims.SessionID,
			appctx.GetTraceID(ctx),
			memberIDs,
			EventChannelUpdated,
			payload,
			now,
		)
		if repoErr != nil {
			return repoErr
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	cacheCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := s.cache.Delete(cacheCtx, updatedChannel.ID); err != nil {
		slog.ErrorContext(cacheCtx, "failed to delete channel cache",
			slog.String("channel_id", updatedChannel.ID.String()),
			slog.Any("error", err),
		)
	}

	if len(systemMessages) > 0 {
		if err := s.messageCache.SetBatch(cacheCtx, updatedChannel.ID, systemMessages); err != nil {
			slog.ErrorContext(cacheCtx, "failed to seed system messages into cache",
				slog.String("channel_id", updatedChannel.ID.String()),
				slog.Int("count", len(systemMessages)),
				slog.Any("error", err),
			)
		}
	}

	if len(updatedPeers) > 0 {
		peerIDs := getMemberIDs(updatedPeers)

		if err := s.memberCache.InvalidateBatch(cacheCtx, updatedChannel.ID, peerIDs); err != nil {
			slog.ErrorContext(cacheCtx, "failed to invalidate updated peer members in cache",
				slog.String("channel_id", updatedChannel.ID.String()),
				slog.Int("count", len(peerIDs)),
				slog.Any("error", err),
			)
		}
	}

	return &UpdateGroupResult{
		Channel:        updatedChannel,
		SystemMessages: systemMessages,
	}, nil
}

func buildUpdateGroupSystemMessages(
	channelID uuid.UUID,
	actorID uuid.UUID,
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
