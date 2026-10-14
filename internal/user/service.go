package user

import (
	"context"
	"log/slog"
	"time"

	"bonfire-api/internal/appctx"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/crypto"
	"bonfire-api/internal/presence"

	"github.com/google/uuid"
)

const anonymizeBatchLimit = 100
const scheduleDeleteGracePeriod = 30 * 24 * time.Hour

type Service struct {
	cache              *Cache
	repo               *Repository
	cachedRepo         *CachedRepository
	authCache          AuthCache
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
	UserID    uuid.UUID
	SessionID uuid.UUID
	Password  string
	NewEmail  string
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

	txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		u, err = s.repo.UpdateEmail(txCtx, p.UserID, p.NewEmail, now)
		if err != nil {
			return err
		}

		event, err := outbox.New(
			EventEmailUpdated,
			outbox.NewMetadata(appctx.GetTraceID(txCtx), &u.ID, &p.SessionID, nil),
			newEmailUpdatedPayload(u.ID, u.Email, u.UpdatedAt),
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

	s.cacheDelete(ctx, u.ID, "email update")

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

	broadcastIDs := s.getBroadcastTargetIDs(ctx, p.UserID)

	now := time.Now()

	txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		u, err = s.repo.UpdateUsername(txCtx, u.ID, p.NewUsername, now)
		if err != nil {
			return err
		}

		event, err := outbox.New(
			EventUsernameUpdated,
			outbox.NewMetadata(appctx.GetTraceID(txCtx), &u.ID, &p.SessionID, broadcastIDs),
			newUsernameUpdatedPayload(u.ID, u.Username, u.UpdatedAt),
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

	s.cacheDelete(ctx, u.ID, "username update")

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

		event, err := outbox.New(
			EventPasswordUpdated,
			outbox.NewMetadata(appctx.GetTraceID(txCtx), &u.ID, &p.SessionID, nil),
			struct{}{},
			now,
		)
		if err != nil {
			return err
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if txErr != nil {
		return txErr
	}

	s.cacheDeleteAndRevokeSessions(ctx, p.UserID, revokedSessionIDs, "password update")

	return nil
}

type UpdatePreferredPresenceParams struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	Presence  *presence.Presence
	Duration  *PreferredPresenceDuration
}

func (s *Service) UpdatePreferredPresence(ctx context.Context, p UpdatePreferredPresenceParams) (*User, error) {
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

	broadcastIDs := s.getBroadcastTargetIDs(ctx, p.UserID)
	dbNow := time.Now()

	txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		u, err = s.repo.UpdatePreferredPresence(txCtx, u.ID, p.Presence, until, dbNow)
		if err != nil {
			return err
		}

		var eff *presence.Presence
		if effP := u.EffectivePresence(dbNow); effP != nil && effP.IsValid() {
			eff = effP
		}

		event, err := outbox.New(
			EventPreferredPresenceUpdated,
			outbox.NewMetadata(appctx.GetTraceID(txCtx), &u.ID, &p.SessionID, broadcastIDs),
			newPreferredPresenceUpdatedPayload(u.ID, eff, u.UpdatedAt),
			dbNow,
		)
		if err != nil {
			return err
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if txErr != nil {
		return nil, txErr
	}

	s.cacheDelete(ctx, p.UserID, "preferred presence update")

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

	broadcastIDs := s.getBroadcastTargetIDs(ctx, p.UserID)

	now := time.Now()

	txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		u, err = s.repo.UpdateProfile(
			txCtx,
			p.UserID,
			p.DisplayName,
			p.Bio,
			p.AvatarURL,
			p.BannerColor,
			now,
		)
		if err != nil {
			return err
		}

		event, err := outbox.New(
			EventProfileUpdated,
			outbox.NewMetadata(appctx.GetTraceID(txCtx), &u.ID, &p.SessionID, broadcastIDs),
			newProfileUpdatedPayload(u.ID, u.DisplayName, u.Bio, u.AvatarURL, u.BannerColor, u.UpdatedAt),
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

	s.cacheDelete(ctx, p.UserID, "profile update")

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

	broadcastIDs := s.getBroadcastTargetIDs(ctx, p.UserID)
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

		event, err := outbox.New(
			EventDisabled,
			outbox.NewMetadata(appctx.GetTraceID(txCtx), &u.ID, &p.SessionID, broadcastIDs),
			newDisabledPayload(u.ID, u.UpdatedAt),
			now,
		)
		if err != nil {
			return err
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if txErr != nil {
		return txErr
	}

	s.cacheDeleteAndRevokeSessions(ctx, p.UserID, revokedSessionIDs, "account disable")

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

	broadcastIDs := s.getBroadcastTargetIDs(ctx, p.UserID)
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

		event, err := outbox.New(
			EventDisabled,
			outbox.NewMetadata(appctx.GetTraceID(txCtx), &u.ID, &p.SessionID, broadcastIDs),
			newDisabledPayload(u.ID, u.UpdatedAt),
			now,
		)
		if err != nil {
			return err
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if txErr != nil {
		return txErr
	}

	s.cacheDeleteAndRevokeSessions(ctx, p.UserID, revokedSessionIDs, "schedule delete")

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

func equalStringPtr(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
