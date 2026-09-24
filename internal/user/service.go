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

type Service struct {
	cache              Cache
	repo               Repository
	cachedRepo         CachedRepository
	cachedRelationRepo CachedRelationRepository
	outboxRepo         OutboxRepository
	sessionRepo        SessionRepository
	tx                 TX
}

func NewService(
	cache Cache,
	repo Repository,
	cachedRepo CachedRepository,
	cachedRelationRepo CachedRelationRepository,
	outboxRepo OutboxRepository,
	sessionRepo SessionRepository,
	tx TX,
) *Service {
	return &Service{
		cache:              cache,
		repo:               repo,
		cachedRepo:         cachedRepo,
		cachedRelationRepo: cachedRelationRepo,
		outboxRepo:         outboxRepo,
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
	Password string
	NewEmail string
}

func (s *Service) UpdateEmail(ctx context.Context, p UpdateEmailParams) (*User, error) {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return nil, err
	}

	u, err := s.fetchAndAuthenticate(ctx, claims.UserID, p.Password)
	if err != nil {
		return nil, err
	}

	if u.Email == p.NewEmail {
		return u, nil
	}

	var updatedUser *User
	now := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var txErr error
		updatedUser, txErr = s.repo.UpdateEmail(txCtx, claims.UserID, p.NewEmail, now)
		if txErr != nil {
			return txErr
		}

		payload := EventEmailUpdatedPayload{
			UserID:    claims.UserID,
			Email:     updatedUser.Email,
			UpdatedAt: now,
		}

		event, txErr := outbox.New(
			claims.UserID,
			claims.SessionID,
			appctx.GetTraceID(ctx),
			[]uuid.UUID{},
			EventEmailUpdated,
			payload,
			now,
		)
		if txErr != nil {
			return txErr
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	if err := s.cache.Delete(ctx, claims.UserID); err != nil {
		slog.WarnContext(ctx, "failed to invalidate user cache after email update",
			slog.String("user_id", claims.UserID.String()),
			slog.Any("error", err),
		)
	}

	return updatedUser, nil
}

type UpdateUsernameParams struct {
	Password    string
	NewUsername string
}

func (s *Service) UpdateUsername(ctx context.Context, p UpdateUsernameParams) (*User, error) {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return nil, err
	}

	u, err := s.fetchAndAuthenticate(ctx, claims.UserID, p.Password)
	if err != nil {
		return nil, err
	}

	if u.Username == p.NewUsername {
		return u, nil
	}

	broadcastIDs := s.getBroadcastTargetIDs(ctx, claims.UserID)

	var updatedUser *User
	now := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var txErr error
		updatedUser, txErr = s.repo.UpdateUsername(txCtx, u.ID, p.NewUsername, now)
		if txErr != nil {
			return txErr
		}

		payload := EventUsernameUpdatedPayload{
			UserID:    claims.UserID,
			Username:  updatedUser.Username,
			UpdatedAt: now,
		}

		event, txErr := outbox.New(
			claims.UserID,
			claims.SessionID,
			appctx.GetTraceID(ctx),
			broadcastIDs,
			EventUsernameUpdated,
			payload,
			now,
		)
		if txErr != nil {
			return txErr
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	if err := s.cache.Delete(ctx, updatedUser.ID); err != nil {
		slog.WarnContext(ctx, "failed to invalidate user cache after username update",
			slog.String("user_id", updatedUser.ID.String()),
			slog.Any("error", err),
		)
	}

	return updatedUser, nil
}

type UpdatePasswordParams struct {
	CurrentPassword    string
	NewPassword        string
	NewPasswordConfirm string
}

func (s *Service) UpdatePassword(ctx context.Context, p UpdatePasswordParams) error {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return err
	}

	if p.NewPassword != p.NewPasswordConfirm {
		return ErrPasswordMismatch()
	}

	u, err := s.fetchAndAuthenticate(ctx, claims.UserID, p.CurrentPassword)
	if err != nil {
		return err
	}

	newPasswordHash, err := crypto.HashPassword(p.NewPassword)
	if err != nil {
		return ErrPasswordHashFailed().Wrap(err)
	}

	now := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var txErr error
		_, txErr = s.repo.UpdatePasswordHash(txCtx, u.ID, newPasswordHash, now)
		if txErr != nil {
			return txErr
		}

		if _, txErr := s.sessionRepo.RevokeAll(txCtx, claims.UserID, now); txErr != nil {
			return txErr
		}

		payload := EventPasswordUpdatedPayload{
			UserID:    claims.UserID,
			UpdatedAt: now,
		}

		event, txErr := outbox.New(
			claims.UserID,
			claims.SessionID,
			appctx.GetTraceID(ctx),
			[]uuid.UUID{},
			EventPasswordUpdated,
			payload,
			now,
		)
		if txErr != nil {
			return txErr
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if err != nil {
		return err
	}

	if err := s.cache.Delete(ctx, claims.UserID); err != nil {
		slog.WarnContext(ctx, "failed to invalidate user cache after password update",
			slog.String("user_id", claims.UserID.String()),
			slog.Any("error", err),
		)
	}

	return nil
}

type UpdatePreferredPresenceParams struct {
	Presence *presence.Presence
	Duration *PreferredPresenceDuration
}

func (s *Service) UpdatePreferredPresence(ctx context.Context, p UpdatePreferredPresenceParams) (*User, error) {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return nil, err
	}

	validateNow := time.Now()

	until, err := p.Duration.CalculateUntil(validateNow)
	if err != nil {
		return nil, err
	}

	u, err := s.Get(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	if u.PreferredPresence == p.Presence {
		return u, nil
	}

	broadcastIDs := s.getBroadcastTargetIDs(ctx, claims.UserID)

	var updatedUser *User
	dbNow := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var txErr error
		updatedUser, txErr = s.repo.UpdatePresence(txCtx, u.ID, p.Presence, until, dbNow)
		if txErr != nil {
			return txErr
		}

		eff := presence.PresenceOnline
		if p := updatedUser.EffectivePresence(dbNow); p != nil && p.IsValid() {
			eff = *p
		}

		payload := EventPreferredPresenceUpdatedPayload{
			UserID:    claims.UserID,
			Presence:  eff,
			UpdatedAt: dbNow,
		}

		event, txErr := outbox.New(
			claims.UserID,
			claims.SessionID,
			appctx.GetTraceID(ctx),
			broadcastIDs,
			EventPreferredPresenceUpdated,
			payload,
			dbNow,
		)
		if txErr != nil {
			return txErr
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	if err := s.cache.Delete(ctx, claims.UserID); err != nil {
		slog.WarnContext(ctx, "failed to invalidate user cache after preferred_presence update",
			slog.String("user_id", claims.UserID.String()),
			slog.Any("error", err),
		)
	}

	return updatedUser, nil
}

type UpdateProfileParams struct {
	DisplayName string
	Bio         *string
	AvatarURL   *string
	BannerColor *string
}

func (s *Service) UpdateProfile(ctx context.Context, p UpdateProfileParams) (*User, error) {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return nil, err
	}

	u, err := s.Get(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	if u.DisplayName == p.DisplayName &&
		equalStringPtr(u.Bio, p.Bio) &&
		equalStringPtr(u.AvatarURL, p.AvatarURL) &&
		equalStringPtr(u.BannerColor, p.BannerColor) {
		return u, nil
	}

	broadcastIDs := s.getBroadcastTargetIDs(ctx, claims.UserID)

	var updatedUser *User
	now := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var txErr error
		updatedUser, txErr = s.repo.UpdateProfile(txCtx, claims.UserID, p.DisplayName, p.Bio, p.AvatarURL, p.BannerColor, now)
		if txErr != nil {
			return txErr
		}

		payload := EventProfileUpdatedPayload{
			UserID:      updatedUser.ID,
			DisplayName: updatedUser.DisplayName,
			Bio:         updatedUser.Bio,
			AvatarURL:   updatedUser.AvatarURL,
			BannerColor: updatedUser.BannerColor,
			UpdatedAt:   updatedUser.UpdatedAt,
		}

		event, txErr := outbox.New(
			claims.UserID,
			claims.SessionID,
			appctx.GetTraceID(ctx),
			broadcastIDs,
			EventProfileUpdated,
			payload,
			now,
		)
		if txErr != nil {
			return txErr
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if err != nil {
		return nil, err
	}

	if err := s.cache.Delete(ctx, updatedUser.ID); err != nil {
		slog.WarnContext(ctx, "failed to invalidate user cache after profile update",
			slog.String("user_id", updatedUser.ID.String()),
			slog.Any("error", err),
		)
	}

	return updatedUser, nil
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

type DisableParams struct {
	Password string
}

func (s *Service) Disable(ctx context.Context, p DisableParams) error {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return err
	}

	u, err := s.fetchAndAuthenticate(ctx, claims.UserID, p.Password)
	if err != nil {
		return err
	}
	if u.IsDisabled() {
		return nil
	}

	targetIDs := s.getBroadcastTargetIDs(ctx, claims.UserID)
	now := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		_, err := s.repo.SetDisabled(txCtx, claims.UserID, &now, now)
		if err != nil {
			return err
		}

		if _, err := s.sessionRepo.RevokeAll(txCtx, claims.UserID, now); err != nil {
			return err
		}

		payload := EventDisabledPayload{
			UserID:    claims.UserID,
			UpdatedAt: now,
		}

		event, txErr := outbox.New(
			claims.UserID,
			claims.SessionID,
			appctx.GetTraceID(ctx),
			targetIDs,
			EventDisabled,
			payload,
			now,
		)
		if txErr != nil {
			return txErr
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if err != nil {
		return err
	}

	if err := s.cache.Delete(ctx, claims.UserID); err != nil {
		slog.WarnContext(ctx, "failed to invalidate user cache after account disable",
			slog.String("user_id", claims.UserID.String()),
			slog.Any("error", err),
		)
	}

	return nil
}

type ScheduleDeleteParams struct {
	Password string
}

func (s *Service) ScheduleDelete(ctx context.Context, p ScheduleDeleteParams) error {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return err
	}

	u, err := s.fetchAndAuthenticate(ctx, claims.UserID, p.Password)
	if err != nil {
		return err
	}

	if u.IsScheduledForDeletion() {
		return nil
	}

	targetIDs := s.getBroadcastTargetIDs(ctx, claims.UserID)
	now := time.Now()
	scheduledAt := now.Add(ScheduleDeleteGracePeriod)

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		_, err := s.repo.SetDeleteSchedule(txCtx, claims.UserID, &scheduledAt, &now, now)
		if err != nil {
			return err
		}

		if _, err := s.sessionRepo.RevokeAll(txCtx, claims.UserID, now); err != nil {
			return err
		}

		payload := EventDisabledPayload{
			UserID:    claims.UserID,
			UpdatedAt: now,
		}

		event, txErr := outbox.New(
			claims.UserID,
			claims.SessionID,
			appctx.GetTraceID(ctx),
			targetIDs,
			EventDisabled,
			payload,
			now,
		)
		if txErr != nil {
			return txErr
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if err != nil {
		return err
	}

	if err := s.cache.Delete(ctx, claims.UserID); err != nil {
		slog.WarnContext(ctx, "failed to invalidate user cache after delete schedule",
			slog.String("user_id", claims.UserID.String()),
			slog.Any("error", err),
		)
	}

	return nil
}

func (s *Service) AnonymizeBatch(ctx context.Context) error {
	now := time.Now()

	users, err := s.repo.ListDeleteScheduled(ctx, now, AnonymizeBatchSize)
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

	if err := s.cache.DeleteBatch(ctx, invalidIDs); err != nil {
		slog.WarnContext(ctx, "failed to invalidate batch user cache after batch anonymize",
			slog.Any("error", err),
		)
	}

	return nil
}

func (s *Service) fetchAndAuthenticate(ctx context.Context, actorID uuid.UUID, password string) (*User, error) {
	u, err := s.fetchValid(ctx, actorID)
	if err != nil {
		return nil, err
	}

	if err := crypto.ComparePasswords(u.PasswordHash, password); err != nil {
		return nil, ErrInvalidPassword().Wrap(err)
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

func (s *Service) fetchBatchValid(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*User, error) {
	usersMap, err := s.repo.GetBatch(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	validUsers := make(map[uuid.UUID]*User, len(usersMap))
	for id, u := range usersMap {
		if u == nil || u.EnsureActive() != nil {
			continue
		}
		validUsers[id] = u
	}

	return validUsers, nil
}

func (s *Service) getBroadcastTargetIDs(ctx context.Context, actorID uuid.UUID) []uuid.UUID {
	friendIDs, err := s.cachedRelationRepo.GetFriendIDs(ctx, actorID)
	if err != nil {
		slog.WarnContext(ctx, "failed to fetch friend IDs for event broadcast, falling back to actor only",
			slog.String("user_id", actorID.String()),
			slog.Any("error", err),
		)
		friendIDs = nil
	}

	targetIDs := make([]uuid.UUID, 0, len(friendIDs)+1)
	targetIDs = append(targetIDs, actorID)
	targetIDs = append(targetIDs, friendIDs...)

	return targetIDs
}
