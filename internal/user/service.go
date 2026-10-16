package user

import (
	"context"
	"log/slog"
	"time"

	"bonfire-api/internal/gateway"
	"bonfire-api/internal/pkg/crypto"

	"github.com/google/uuid"
)

const anonymizeBatchLimit = 100
const scheduleDeleteGracePeriod = 30 * 24 * time.Hour

type Service struct {
	cache              *Cache
	repo               *Repository
	cachedRepo         *CachedRepository
	authCache          AuthCache
	broadcaster        Broadcaster
	outboxRepo         OutboxRepository
	cachedRelationRepo CachedRelationRepository
	sessionCache       SessionCache
	sessionRepo        SessionRepository
	tx                 TX
}

func NewService(
	cache *Cache,
	repo *Repository,
	cachedRepo *CachedRepository,
	authCache AuthCache,
	broadcaster Broadcaster,
	outboxRepo OutboxRepository,
	cachedRelationRepo CachedRelationRepository,
	sessionCache SessionCache,
	sessionRepo SessionRepository,
	tx TX,
) *Service {
	return &Service{
		cache:              cache,
		repo:               repo,
		cachedRepo:         cachedRepo,
		authCache:          authCache,
		broadcaster:        broadcaster,
		outboxRepo:         outboxRepo,
		cachedRelationRepo: cachedRelationRepo,
		sessionCache:       sessionCache,
		sessionRepo:        sessionRepo,
		tx:                 tx,
	}
}

func (s *Service) Get(ctx context.Context, userID uuid.UUID) (*User, error) {
	u, err := s.cachedRepo.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	return u, nil
}

type UpdateEmailParams struct {
	UserID   uuid.UUID
	Password string
	NewEmail string
}

func (s *Service) UpdateEmail(ctx context.Context, p UpdateEmailParams) (*User, error) {
	u, err := s.fetchAndAuthenticate(ctx, p.UserID, p.Password)
	if err != nil {
		return nil, err
	}

	if u.Email == p.NewEmail {
		return u, nil
	}

	now := time.Now()

	u, err = s.repo.UpdateEmail(ctx, p.UserID, p.NewEmail, now)
	if err != nil {
		return nil, err
	}

	s.cacheDelete(ctx, u.ID, "email update")

	s.broadcastToUsers(
		ctx,
		u.ID,
		[]uuid.UUID{u.ID},
		MessageEmailUpdated,
		newEmailUpdatedPayload(u.ID, u.Email, u.UpdatedAt),
		"failed to broadcast email updated",
	)

	return u, nil
}

type UpdateUsernameParams struct {
	UserID      uuid.UUID
	SessionID   uuid.UUID
	Password    string
	NewUsername string
}

func (s *Service) UpdateUsername(ctx context.Context, p UpdateUsernameParams) (*User, error) {
	u, err := s.fetchAndAuthenticate(ctx, p.UserID, p.Password)
	if err != nil {
		return nil, err
	}

	if u.Username == p.NewUsername {
		return u, nil
	}

	now := time.Now()

	u, err = s.repo.UpdateUsername(ctx, u.ID, p.NewUsername, now)
	if err != nil {
		return nil, err
	}

	s.cacheDelete(ctx, u.ID, "username update")

	s.broadcastToUsers(
		ctx,
		u.ID,
		s.getBroadcastTargetIDs(ctx, p.UserID),
		MessageUsernameUpdated,
		newUsernameUpdatedPayload(u.ID, u.Username, u.UpdatedAt),
		"failed to broadcast username updated",
	)

	return u, nil
}

type UpdatePasswordParams struct {
	UserID             uuid.UUID
	SessionID          uuid.UUID
	CurrentPassword    string
	NewPassword        string
	NewPasswordConfirm string
}

func (s *Service) UpdatePassword(ctx context.Context, p UpdatePasswordParams) error {
	if p.NewPassword != p.NewPasswordConfirm {
		return ErrPasswordMismatch()
	}

	u, err := s.fetchAndAuthenticate(ctx, p.UserID, p.CurrentPassword)
	if err != nil {
		return err
	}

	newPasswordHash, err := crypto.HashPassword(p.NewPassword)
	if err != nil {
		return err
	}

	var revokedSessionIDs []uuid.UUID
	now := time.Now()

	txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		u, err = s.repo.UpdatePasswordHash(txCtx, u.ID, newPasswordHash, now)
		if err != nil {
			return err
		}

		revokedSessionIDs, err = s.sessionRepo.RevokeAllExcept(txCtx, u.ID, p.SessionID, now)
		if err != nil {
			return err
		}

		return nil
	})
	if txErr != nil {
		return txErr
	}

	s.cacheDeleteAndRevokeSessions(ctx, p.UserID, revokedSessionIDs, "password update")

	s.broadcastToUserSessions(
		ctx,
		p.UserID,
		revokedSessionIDs,
		MessagePasswordUpdated,
		nil,
		"failed to broadcast password updated",
	)

	return nil
}

type UpdatePreferredPresenceParams struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	Presence  *gateway.Presence
	Duration  *PreferredPresenceDuration
}

func (s *Service) UpdatePreferredPresence(ctx context.Context, p UpdatePreferredPresenceParams) (*User, error) {
	isPreferred := p.Presence == nil || p.Presence.IsPreferred()
	if !isPreferred {
		return nil, ErrPreferredPresenceInvalid()
	}

	validateNow := time.Now()

	var until *time.Time
	if p.Duration != nil {
		var err error
		until, err = p.Duration.CalculateUntil(validateNow)
		if err != nil {
			return nil, err
		}
	}

	u, err := s.Get(ctx, p.UserID)
	if err != nil {
		return nil, err
	}

	presenceEqual := (u.PreferredPresence == nil && p.Presence == nil) ||
		(u.PreferredPresence != nil && p.Presence != nil && *u.PreferredPresence == *p.Presence)

	expiresAtEqual := (u.PreferredPresenceUntil == nil && until == nil) ||
		(u.PreferredPresenceUntil != nil && until != nil && u.PreferredPresenceUntil.Equal(*until))

	if presenceEqual && expiresAtEqual {
		return u, nil
	}

	dbNow := time.Now()

	u, err = s.repo.UpdatePreferredPresence(ctx, u.ID, p.Presence, until, dbNow)
	if err != nil {
		return nil, err
	}

	s.cacheDelete(ctx, p.UserID, "preferred presence update")

	s.broadcastToUsers(
		ctx,
		u.ID,
		s.getBroadcastTargetIDs(ctx, p.UserID),
		MessagePreferredPresenceUpdated,
		newPreferredPresenceUpdatedPayload(u.ID, u.EffectivePresence(dbNow), u.UpdatedAt),
		"failed to broadcast preferred presence updated",
	)

	return u, nil
}

type UpdateProfileParams struct {
	UserID      uuid.UUID
	SessionID   uuid.UUID
	DisplayName string
	Bio         *string
	AvatarURL   *string
	BannerColor *string
}

func (s *Service) UpdateProfile(ctx context.Context, p UpdateProfileParams) (*User, error) {
	u, err := s.Get(ctx, p.UserID)
	if err != nil {
		return nil, err
	}

	if u.DisplayName == p.DisplayName &&
		equalStringPtr(u.Bio, p.Bio) &&
		equalStringPtr(u.AvatarURL, p.AvatarURL) &&
		equalStringPtr(u.BannerColor, p.BannerColor) {
		return u, nil
	}

	now := time.Now()

	u, err = s.repo.UpdateProfile(
		ctx,
		p.UserID,
		p.DisplayName,
		p.Bio,
		p.AvatarURL,
		p.BannerColor,
		now,
	)
	if err != nil {
		return nil, err
	}

	s.cacheDelete(ctx, p.UserID, "profile update")

	s.broadcastToUsers(
		ctx,
		u.ID,
		s.getBroadcastTargetIDs(ctx, p.UserID),
		MessageProfileUpdated,
		newProfileUpdatedPayload(u.ID, u.DisplayName, u.Bio, u.AvatarURL, u.BannerColor, u.UpdatedAt),
		"failed to broadcast profile updated",
	)

	return u, nil
}

type DisableParams struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	Password  string
}

func (s *Service) Disable(ctx context.Context, p DisableParams) error {
	u, err := s.fetchAndAuthenticate(ctx, p.UserID, p.Password)
	if err != nil {
		return err
	}
	if u.IsDisabled() {
		return nil
	}

	var revokedSessionIDs []uuid.UUID
	now := time.Now()

	txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		u, err = s.repo.SetDisabled(txCtx, p.UserID, &now, now)
		if err != nil {
			return err
		}

		revokedSessionIDs, err = s.sessionRepo.RevokeAll(txCtx, p.UserID, now)
		if err != nil {
			return err
		}

		return nil
	})
	if txErr != nil {
		return txErr
	}

	s.cacheDeleteAndRevokeSessions(ctx, p.UserID, revokedSessionIDs, "account disable")

	s.broadcastToUsers(
		ctx,
		u.ID,
		s.getBroadcastTargetIDs(ctx, p.UserID),
		MessageDisabled,
		newDisabledPayload(u.ID, u.UpdatedAt),
		"failed to broadcast disabled",
	)

	return nil
}

type ScheduleDeleteParams struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	Password  string
}

func (s *Service) ScheduleDelete(ctx context.Context, p ScheduleDeleteParams) error {
	u, err := s.fetchAndAuthenticate(ctx, p.UserID, p.Password)
	if err != nil {
		return err
	}
	if u.IsScheduledForDeletion() {
		return nil
	}

	var revokedSessionIDs []uuid.UUID
	now := time.Now()
	scheduledAt := now.Add(scheduleDeleteGracePeriod)

	txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		u, err = s.repo.SetDeleteSchedule(txCtx, p.UserID, &scheduledAt, &now, now)
		if err != nil {
			return err
		}

		revokedSessionIDs, err = s.sessionRepo.RevokeAll(txCtx, p.UserID, now)
		if err != nil {
			return err
		}

		return nil
	})
	if txErr != nil {
		return txErr
	}

	s.cacheDeleteAndRevokeSessions(ctx, p.UserID, revokedSessionIDs, "schedule delete")

	s.broadcastToUsers(
		ctx,
		u.ID,
		s.getBroadcastTargetIDs(ctx, p.UserID),
		MessageDisabled,
		newDisabledPayload(u.ID, u.UpdatedAt),
		"failed to broadcast schedule delete",
	)

	return nil
}

func (s *Service) AnonymizeBatch(ctx context.Context) error {
	now := time.Now()

	users, err := s.repo.ListDeleteScheduled(ctx, now, anonymizeBatchLimit)
	if err != nil {
		return err
	}

	if len(users) == 0 {
		return nil
	}

	for _, u := range users {
		u.Anonymize(now)
	}

	_, err = s.repo.UpdateBatch(ctx, users)
	if err != nil {
		return err
	}

	invalidIDs := make([]uuid.UUID, len(users))
	for i, u := range users {
		invalidIDs[i] = u.ID
	}

	s.cacheDeleteBatch(ctx, invalidIDs, "batch anonymize")

	return nil
}

func (s *Service) fetchAndAuthenticate(ctx context.Context, actorID uuid.UUID, password string) (*User, error) {
	u, err := s.fetchValid(ctx, actorID)
	if err != nil {
		return nil, err
	}

	if err := crypto.CompareHashAndPassword(u.PasswordHash, password); err != nil {
		return nil, err
	}

	return u, nil
}

func (s *Service) fetchValid(ctx context.Context, actorID uuid.UUID) (*User, error) {
	u, err := s.repo.Get(ctx, actorID)
	if err != nil {
		return nil, err
	}

	if err := u.EnsureActive(); err != nil {
		return nil, err
	}

	return u, nil
}

func (s *Service) getBroadcastTargetIDs(ctx context.Context, actorID uuid.UUID) []uuid.UUID {
	// friendIDs, err := s.cachedRelationRepo.GetFriendIDs(ctx, actorID)
	// if err != nil {
	// 	slog.WarnContext(ctx, "failed to fetch friend IDs for event broadcast, falling back to actor only",
	// 		slog.String("user_id", actorID.String()),
	// 		slog.Any("error", err),
	// 	)
	// 	friendIDs = nil
	// }

	// targetIDs := make([]uuid.UUID, 0, len(friendIDs)+1)
	// targetIDs = append(targetIDs, actorID)
	// targetIDs = append(targetIDs, friendIDs...)

	// return targetIDs

	return []uuid.UUID{actorID}
}

func (s *Service) cacheDelete(ctx context.Context, userID uuid.UUID, action string) {
	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.cache.Delete(cacheCtx, userID); err != nil {
		slog.WarnContext(cacheCtx, "failed to invalidate user cache after "+action,
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
	}
}

func (s *Service) cacheDeleteBatch(ctx context.Context, userIDs []uuid.UUID, action string) {
	if len(userIDs) == 0 {
		return
	}

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.cache.DeleteBatch(cacheCtx, userIDs); err != nil {
		slog.WarnContext(cacheCtx, "failed to invalidate user batch cache after "+action,
			slog.Int("user_count", len(userIDs)),
			slog.Any("error", err),
		)
	}
}

func (s *Service) cacheDeleteAndRevokeSessions(ctx context.Context, userID uuid.UUID, sessionIDs []uuid.UUID, action string) {
	count := len(sessionIDs)
	if count == 0 {
		return
	}

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.authCache.DeleteUserAndRevokeSessions(cacheCtx, userID, sessionIDs); err != nil {
		slog.WarnContext(cacheCtx, "failed to invalidate user and session cache after "+action,
			slog.String("user_id", userID.String()),
			slog.Int("revoked_session_count", count),
			slog.Any("error", err),
		)
	}
}

func (s *Service) broadcastEmailUpdated(ctx context.Context, p MessageEmailUpdatedPayload) {
	bCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.broadcaster.ToUser(bCtx, p.UserID, MessageEmailUpdated, p); err != nil {
		slog.WarnContext(bCtx, "failed to broadcast email updated",
			slog.String("user_id", p.UserID.String()),
			slog.Any("error", err),
		)
	}
}

func (s *Service) broadcastUsernameUpdated(ctx context.Context, recipientIDs []uuid.UUID, p MessageUsernameUpdatedPayload) {
	if len(recipientIDs) == 0 {
		return
	}

	bCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.broadcaster.ToUsers(bCtx, recipientIDs, MessageUsernameUpdated, p); err != nil {
		slog.WarnContext(bCtx, "failed to broadcast username updated",
			slog.String("user_id", p.UserID.String()),
			slog.Any("error", err),
		)
	}
}

func (s *Service) broadcastPasswordUpdated(ctx context.Context, uid uuid.UUID, revokedSessionIDs []uuid.UUID) {
	if len(revokedSessionIDs) == 0 {
		return
	}

	bCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.broadcaster.ToUserSessions(bCtx, uid, revokedSessionIDs, MessagePasswordUpdated, nil); err != nil {
		slog.WarnContext(bCtx, "failed to broadcast password updated",
			slog.String("user_id", uid.String()),
			slog.Any("error", err),
		)
	}
}

func (s *Service) broadcastPreferredPresenceUpdated(ctx context.Context, recipientIDs []uuid.UUID, p MessagePreferredPresenceUpdatedPayload) {
	if len(recipientIDs) == 0 {
		return
	}

	bCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.broadcaster.ToUsers(bCtx, recipientIDs, MessagePreferredPresenceUpdated, p); err != nil {
		slog.WarnContext(bCtx, "failed to broadcast preferred presence updated",
			slog.String("user_id", p.UserID.String()),
			slog.Any("error", err),
		)
	}
}

func (s *Service) broadcastProfileUpdated(ctx context.Context, recipientIDs []uuid.UUID, p MessageProfileUpdatedPayload) {
	if len(recipientIDs) == 0 {
		return
	}

	bCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.broadcaster.ToUsers(bCtx, recipientIDs, MessageProfileUpdated, p); err != nil {
		slog.WarnContext(bCtx, "failed to broadcast profile updated",
			slog.String("user_id", p.UserID.String()),
			slog.Any("error", err),
		)
	}
}

func (s *Service) broadcastDisabled(ctx context.Context, recipientIDs []uuid.UUID, p MessageDisabledPayload) {
	if len(recipientIDs) == 0 {
		return
	}

	bCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.broadcaster.ToUsers(bCtx, recipientIDs, MessageDisabled, p); err != nil {
		slog.WarnContext(bCtx, "failed to broadcast disabled",
			slog.String("user_id", p.UserID.String()),
			slog.Any("error", err),
		)
	}
}

func (s *Service) broadcastToUsers(ctx context.Context, userID uuid.UUID, recipientIDs []uuid.UUID, messageType string, payload any, logMsg string) {
	if len(recipientIDs) == 0 {
		return
	}

	bCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.broadcaster.ToUsers(bCtx, recipientIDs, messageType, payload); err != nil {
		slog.WarnContext(bCtx, logMsg,
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
	}
}

func (s *Service) broadcastToUserSessions(ctx context.Context, userID uuid.UUID, sessionIDs []uuid.UUID, messageType string, payload any, logMsg string) {
	if len(sessionIDs) == 0 {
		return
	}

	bCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.broadcaster.ToUserSessions(bCtx, userID, sessionIDs, messageType, payload); err != nil {
		slog.WarnContext(bCtx, logMsg,
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
	}
}

func equalStringPtr(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
