package user

import (
	"context"
	"encoding/json"
	"log/slog"

	"bonfire-api/internal/crypto"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/pubsub"

	"github.com/google/uuid"
)

type Service struct {
	repo       Repository
	cache      Cache
	outboxRepo OutboxRepository
	tx         TX
	gatewayPub GatewayPub
}

func NewService(
	repo Repository,
	cache Cache,
	outboxRepo OutboxRepository,
	tx TX,
	gatewayPub GatewayPub,
) *Service {
	return &Service{
		repo:       repo,
		cache:      cache,
		outboxRepo: outboxRepo,
		tx:         tx,
		gatewayPub: gatewayPub,
	}
}

// -----------------------------------------------------------------------------
// Get
// -----------------------------------------------------------------------------

func (s *Service) Get(ctx context.Context, userID uuid.UUID) (*User, error) {
	id, err := fields.ParseRequiredID("id", userID)
	if err != nil {
		return nil, err
	}

	if u := s.GetCache(ctx, id); u != nil {
		if err := u.EnsureActive(); err != nil {
			s.cache.Delete(ctx, id)
			return nil, err
		}
		return u, nil
	}

	u, err := s.fetchValid(ctx, id)
	if err != nil {
		return nil, err
	}

	s.SetCache(ctx, u)

	return u, nil
}

// -----------------------------------------------------------------------------
// Bootstrap Helpers
// -----------------------------------------------------------------------------

func (s *Service) GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*User, error) {
	if len(ids) == 0 {
		return make(map[fields.ID]*User), nil
	}

	found, missing := s.GetBatchCache(ctx, ids)

	validUsers := make(map[fields.ID]*User, len(ids))
	var invalidIDs []fields.ID

	for id, u := range found {
		if u == nil {
			missing = append(missing, id)
			continue
		}

		if err := u.EnsureActive(); err == nil {
			validUsers[id] = u
		} else {
			invalidIDs = append(invalidIDs, id)
		}
	}

	if len(invalidIDs) > 0 {
		s.cache.DeleteBatch(ctx, invalidIDs)
	}

	if len(missing) == 0 {
		return validUsers, nil
	}

	missing = fields.DedupeIDs(missing)

	dbUsersMap, err := s.fetchBatchValid(ctx, missing)
	if err != nil {
		return nil, err
	}

	if len(dbUsersMap) == 0 {
		return validUsers, nil
	}

	s.SetBatchCache(ctx, dbUsersMap)

	for id, u := range dbUsersMap {
		validUsers[id] = u
	}

	return validUsers, nil
}

func (s *Service) GetBatchPresence(ctx context.Context, userIDs []fields.ID) (map[fields.ID]Presence, error) {
	return s.cache.GetBatchPresence(ctx, userIDs)
}

// -----------------------------------------------------------------------------
// Account Management
// -----------------------------------------------------------------------------

type UpdateEmailParams struct {
	UserID   uuid.UUID
	NewEmail string
	Password string
}

func (s *Service) UpdateEmail(ctx context.Context, p UpdateEmailParams) (*User, error) {
	newEmail, err := ParseEmail("email", p.NewEmail)
	if err != nil {
		return nil, err
	}

	id, u, err := s.parseAndAuthenticate(ctx, p.UserID, p.Password)
	if err != nil {
		return nil, err
	}

	if u.email.Equals(newEmail.Text) {
		return u, nil
	}

	now := fields.Now()

	updatedUser, err := s.repo.UpdateEmail(ctx, id, newEmail, now)
	if err != nil {
		return nil, err
	}

	s.cache.Delete(ctx, updatedUser.ID())

	return updatedUser, nil
}

type UpdateUsernameParams struct {
	UserID      uuid.UUID
	NewUsername string
	Password    string
}

func (s *Service) UpdateUsername(ctx context.Context, p UpdateUsernameParams) (*User, error) {
	newUsername, err := ParseUsername("username", p.NewUsername)
	if err != nil {
		return nil, err
	}

	id, u, err := s.parseAndAuthenticate(ctx, p.UserID, p.Password)
	if err != nil {
		return nil, err
	}

	if u.username.Equals(newUsername.Text) {
		return u, nil
	}

	now := fields.Now()
	var updatedUser *User

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var txErr error
		updatedUser, txErr = s.repo.UpdateUsername(txCtx, id, newUsername, now)
		if txErr != nil {
			return txErr
		}

		payload := EventUpdateUsernamePayload{
			UserID:    updatedUser.ID().String(),
			Username:  updatedUser.Username().String(),
			UpdatedAt: updatedUser.UpdatedAt().String(),
		}

		return s.outboxRepo.Publish(txCtx, EventUpdateUsername, payload, now)
	})
	if err != nil {
		return nil, err
	}

	s.cache.Delete(ctx, updatedUser.ID())

	return updatedUser, nil
}

type UpdatePasswordParams struct {
	UserID             uuid.UUID
	CurrentPassword    string
	NewPassword        string
	NewPasswordConfirm string
}

func (s *Service) UpdatePassword(ctx context.Context, p UpdatePasswordParams) error {
	currentPassword, err := ParsePassword("current_password", p.CurrentPassword)
	if err != nil {
		return err
	}

	newPassword, err := ParsePassword("new_password", p.NewPassword)
	if err != nil {
		return err
	}

	newPasswordConfirm, err := ParsePassword("new_password_confirm", p.NewPasswordConfirm)
	if err != nil {
		return err
	}

	if !newPassword.Equals(newPasswordConfirm.Text) {
		return ErrPasswordMismatch("new_password_confirm")
	}

	id, _, err := s.parseAndAuthenticate(ctx, p.UserID, currentPassword.String())
	if err != nil {
		return err
	}

	passwordHash, err := crypto.HashPassword(newPassword.String())
	if err != nil {
		return ErrPasswordHashFailed(err)
	}

	newPasswordHash, err := ParsePasswordHash("new_password_hash", passwordHash)
	if err != nil {
		return err
	}

	now := fields.Now()

	_, err = s.repo.UpdatePasswordHash(ctx, id, newPasswordHash, now)
	if err != nil {
		return err
	}

	s.cache.Delete(ctx, id)

	return nil
}

type UpdatePreferredPresenceParams struct {
	UserID   uuid.UUID
	Presence *string
	Duration *string
}

func (s *Service) UpdatePreferredPresence(ctx context.Context, p UpdatePreferredPresenceParams) (*User, error) {
	id, err := fields.ParseRequiredID("id", p.UserID)
	if err != nil {
		return nil, err
	}

	preferredPresence, err := ParsePreferredPresence("preferred_presence", ptr.From(p.Presence))
	if err != nil {
		return nil, err
	}

	duration, err := ParsePreferredPresenceDurationString(ptr.From(p.Duration))
	if err != nil {
		return nil, err
	}

	now := fields.Now()
	until, err := duration.CalculateUntil(now)
	if err != nil {
		return nil, err
	}

	u, err := s.Get(ctx, id.UUID())
	if err != nil {
		return nil, err
	}

	if u.PreferredPresence().Equals(preferredPresence) && u.PreferredPresenceUntil().Equals(until) {
		return u, nil
	}

	var updatedUser *User

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var txErr error
		updatedUser, txErr = s.repo.UpdatePresence(txCtx, id, preferredPresence, until, now)
		if txErr != nil {
			return txErr
		}

		effectivePresence := updatedUser.EffectivePresence(now).Presence()
		if !effectivePresence.IsValid() {
			effectivePresence = NewPresenceOnline()
		}

		payload := EventUpdatePresencePayload{
			UserID:    updatedUser.ID().String(),
			Presence:  effectivePresence.String(),
			UpdatedAt: updatedUser.UpdatedAt().String(),
		}

		return s.outboxRepo.Publish(txCtx, EventUpdatePresence, payload, now)
	})
	if err != nil {
		return nil, err
	}

	s.cache.Delete(ctx, updatedUser.ID())

	return updatedUser, nil
}

type UpdateProfileParams struct {
	UserID      uuid.UUID
	DisplayName string
	Bio         *string
	AvatarURL   *string
	BannerColor *string
}

func (s *Service) UpdateProfile(ctx context.Context, p UpdateProfileParams) (*User, error) {
	id, err := fields.ParseRequiredID("id", p.UserID)
	if err != nil {
		return nil, err
	}

	displayName, err := ParseDisplayName("display_name", p.DisplayName)
	if err != nil {
		return nil, err
	}

	bio, err := ParseBio("bio", ptr.From(p.Bio))
	if err != nil {
		return nil, err
	}

	avatarURL, err := fields.ParseURL("avatar_url", ptr.From(p.AvatarURL))
	if err != nil {
		return nil, err
	}

	bannerColor, err := fields.ParseHexColor("banner_color", ptr.From(p.BannerColor))
	if err != nil {
		return nil, err
	}

	u, err := s.Get(ctx, id.UUID())
	if err != nil {
		return nil, err
	}

	if u.DisplayName().Equals(displayName.Text) &&
		u.Bio().Equals(bio.Text) &&
		u.AvatarURL().Equals(avatarURL) &&
		u.BannerColor().Equals(bannerColor.Text) {
		return u, nil
	}

	now := fields.Now()
	var updatedUser *User

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var txErr error
		updatedUser, txErr = s.repo.UpdateProfile(txCtx, id, displayName, bio, avatarURL, bannerColor, now)
		if txErr != nil {
			return txErr
		}

		payload := EventUpdateProfilePayload{
			UserID:      updatedUser.ID().String(),
			DisplayName: updatedUser.DisplayName().String(),
			Bio:         updatedUser.Bio().StringPtr(),
			AvatarURL:   updatedUser.AvatarURL().StringPtr(),
			BannerColor: updatedUser.BannerColor().StringPtr(),
			UpdatedAt:   updatedUser.UpdatedAt().String(),
		}

		return s.outboxRepo.Publish(txCtx, EventUpdateProfile, payload, now)
	})
	if err != nil {
		return nil, err
	}

	s.cache.Delete(ctx, updatedUser.ID())

	return updatedUser, nil
}

type DisableParams struct {
	UserID   uuid.UUID
	Password string
}

func (s *Service) Disable(ctx context.Context, p DisableParams) error {
	id, u, err := s.parseAndAuthenticate(ctx, p.UserID, p.Password)
	if err != nil {
		return err
	}
	if u.IsDisabled() {
		return nil
	}

	now := fields.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		_, err := s.repo.SetDisabled(txCtx, id, now, now)
		if err != nil {
			return err
		}

		payload := EventDisablePayload{
			UserID:    id.String(),
			UpdatedAt: now.String(),
		}

		return s.outboxRepo.Publish(txCtx, EventDisable, payload, now)
	})
	if err != nil {
		return err
	}

	s.cache.Delete(ctx, id)

	return nil
}

type ScheduleDeleteParams struct {
	UserID   uuid.UUID
	Password string
}

func (s *Service) ScheduleDelete(ctx context.Context, p ScheduleDeleteParams) error {
	id, u, err := s.parseAndAuthenticate(ctx, p.UserID, p.Password)
	if err != nil {
		return err
	}

	if u.IsScheduledForDeletion() {
		return nil
	}

	now := fields.Now()
	scheduledAt := fields.NewTimestamp(now.Time().Add(ScheduleDeleteGracePeriod))

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		_, err := s.repo.SetDeleteSchedule(txCtx, id, scheduledAt, now, now)
		if err != nil {
			return err
		}

		payload := EventDisablePayload{
			UserID:    id.String(),
			UpdatedAt: now.String(),
		}

		return s.outboxRepo.Publish(txCtx, EventDisable, payload, now)
	})
	if err != nil {
		return err
	}

	s.cache.Delete(ctx, id)

	return nil
}

func (s *Service) AnonymizeBatch(ctx context.Context) error {
	now := fields.Now()

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
	return err
}

// -----------------------------------------------------------------------------
// Real-Time & Presence
// -----------------------------------------------------------------------------

func (s *Service) RegisterWSConnection(ctx context.Context, userID, nodeID fields.ID, presence Presence) error {
	// 1. Fetch current presence state from cache
	currentPresence, err := s.cache.GetPresence(ctx, userID)
	if err != nil {
		currentPresence = NewPresenceOffline()
	}

	// 2. Resolve the desired presence: default to current if valid, else fallback to Online
	p := presence
	if !p.IsValid() {
		if currentPresence.IsValid() && !currentPresence.IsOffline() {
			p = currentPresence // Preserve existing status (e.g., Busy / DND) for new tabs
		} else {
			p = NewPresenceOnline()
		}
	}

	// 3. Register the connection node in cache
	if err := s.cache.RegisterWSConnection(ctx, userID, nodeID, p); err != nil {
		return err
	}

	// 4. Broadcast ONLY if user transitioned from offline OR changed status
	if currentPresence.IsOffline() || currentPresence != p {
		s.pubUpdatePresence(ctx, userID, p)
	}

	return nil
}

func (s *Service) UnregisterWSConnection(ctx context.Context, userID, nodeID fields.ID) error {
	wentOffline, err := s.cache.UnregisterWSConnection(ctx, userID, nodeID)
	if err != nil {
		return err
	}

	if wentOffline {
		offlinePresence := NewPresenceOffline()
		s.pubUpdatePresence(ctx, userID, offlinePresence)
	}

	return nil
}

func (s *Service) RemoveBatchNode(ctx context.Context, userIDs []fields.ID, nodeID fields.ID) error {
	if len(userIDs) == 0 {
		return nil
	}
	return s.cache.RemoveBatchNode(ctx, userIDs, nodeID)
}

func (s *Service) HandleHeartbeat(ctx context.Context, userID, nodeID fields.ID, newPresence Presence) error {
	currentPresence, err := s.cache.GetPresence(ctx, userID)
	if err != nil {
		return s.RegisterWSConnection(ctx, userID, nodeID, newPresence)
	}

	p := newPresence
	if !p.IsValid() {
		p = currentPresence
	}

	if currentPresence != p {
		return s.RegisterWSConnection(ctx, userID, nodeID, p)
	}

	return s.cache.Heartbeat(ctx, userID, nodeID)
}

func (s *Service) PublishBatch(ctx context.Context, nodeEvents map[fields.ID]pubsub.NodeEvent) error {
	return s.gatewayPub.PublishBatchNodeEvents(ctx, nodeEvents)
}

func (s *Service) pubUpdatePresence(ctx context.Context, userID fields.ID, p Presence) {
	nodeToUsers, err := s.cache.GetUpdateRecipientNodes(ctx, userID)
	if err != nil || len(nodeToUsers) == 0 {
		return
	}

	if len(nodeToUsers) == 0 {
		return
	}

	payload := EventUpdatePresencePayload{
		UserID:   userID.String(),
		Presence: p.String(),
	}

	if err := s.publishToNodes(ctx, nodeToUsers, EventUpdatePresence, payload); err != nil {
		slog.ErrorContext(ctx, "failed to broadcast presence updates to nodes",
			"user_id", userID,
			"error", err,
		)
	}
}

func (s *Service) pubUpdateUsername(ctx context.Context, userID fields.ID, newUsername Username, now fields.Timestamp) {
	nodeToUsers, err := s.cache.GetFriendNodes(ctx, userID)
	if err != nil || len(nodeToUsers) == 0 {
		return
	}

	payload := EventUpdateUsernamePayload{
		UserID:      userID.String(),
		NewUsername: newUsername.String(),
		UpdatedAt:   now.String(),
	}

	if err := s.publishToNodes(ctx, nodeToUsers, EventUpdateUsername, payload); err != nil {
		slog.ErrorContext(ctx, "failed to broadcast username updates", "user_id", userID, "error", err)
	}
}

func (s *Service) pubUpdateProfile(ctx context.Context, updatedUser *User) {
	nodeToUsers, err := s.cache.GetUpdateRecipientNodes(ctx, updatedUser.ID())
	if err != nil || len(nodeToUsers) == 0 {
		return
	}

	payload := EventUpdateProfilePayload{
		UserID:      updatedUser.ID().String(),
		DisplayName: updatedUser.DisplayName().String(),
		Bio:         updatedUser.Bio().StringPtr(),
		AvatarURL:   updatedUser.AvatarURL().StringPtr(),
		BannerColor: updatedUser.BannerColor().StringPtr(),
		UpdatedAt:   updatedUser.UpdatedAt().String(),
	}

	if err := s.publishToNodes(ctx, nodeToUsers, EventUpdateProfile, payload); err != nil {
		slog.ErrorContext(ctx, "failed to broadcast profile updates", "user_id", updatedUser.ID(), "error", err)
	}
}

func (s *Service) pubDisable(ctx context.Context, userID fields.ID, now fields.Timestamp) {
	nodeToUsers, err := s.cache.GetFriendNodes(ctx, userID)
	if err != nil || len(nodeToUsers) == 0 {
		return
	}

	payload := EventDisablePayload{
		UserID:    userID.String(),
		UpdatedAt: now.String(),
	}

	if err := s.publishToNodes(ctx, nodeToUsers, EventDisable, payload); err != nil {
		slog.ErrorContext(ctx, "failed to broadcast disable updates", "user_id", userID, "error", err)
	}
}

func (s *Service) publishToNodes(
	ctx context.Context,
	nodeToUsers map[fields.ID][]fields.ID,
	eventType string,
	payload interface{},
) error {
	if len(nodeToUsers) == 0 {
		return nil
	}

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return errs.Internal("Failed to marshal event payload.").Wrap(err)
	}

	nodeEvents := make(map[fields.ID]pubsub.NodeEvent, len(nodeToUsers))
	for nodeID, targetUserIDs := range nodeToUsers {
		nodeEvents[nodeID] = pubsub.NodeEvent{
			UserIDs: fields.UUIDs(targetUserIDs),
			Type:    eventType,
			Data:    rawPayload,
		}
	}

	return s.PublishBatch(ctx, nodeEvents)
}

// -----------------------------------------------------------------------------
// Cache Helpers
// -----------------------------------------------------------------------------

func (s *Service) GetCache(ctx context.Context, userID fields.ID) *User {
	u, err := s.cache.Get(ctx, userID)
	if err != nil {
		return nil
	}
	return u
}

func (s *Service) GetBatchCache(ctx context.Context, ids []fields.ID) (map[fields.ID]*User, []fields.ID) {
	found, missing, err := s.cache.GetBatch(ctx, ids)
	if err != nil {
		return make(map[fields.ID]*User), ids
	}
	return found, missing
}

func (s *Service) SetCache(ctx context.Context, user *User) {
	s.cache.Set(ctx, user)
}

func (s *Service) SetBatchCache(ctx context.Context, users map[fields.ID]*User) {
	s.cache.SetBatch(ctx, users)
}

// -----------------------------------------------------------------------------
// Internal Helpers
// -----------------------------------------------------------------------------

func (s *Service) fetchValid(ctx context.Context, actorID fields.ID) (*User, error) {
	u, err := s.repo.Get(ctx, actorID)
	if err != nil {
		return nil, err
	}

	if err := u.EnsureActive(); err != nil {
		return nil, err
	}

	return u, nil
}

func (s *Service) fetchBatchValid(ctx context.Context, userIDs []fields.ID) (map[fields.ID]*User, error) {
	if len(userIDs) == 0 {
		return make(map[fields.ID]*User), nil
	}

	usersMap, err := s.repo.GetBatch(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	validUsers := make(map[fields.ID]*User, len(usersMap))
	for id, u := range usersMap {
		if u == nil || u.EnsureActive() != nil {
			continue
		}
		validUsers[id] = u
	}

	return validUsers, nil
}

func (s *Service) fetchAndAuthenticate(ctx context.Context, actorID fields.ID, password Password) (*User, error) {
	u, err := s.fetchValid(ctx, actorID)
	if err != nil {
		return nil, err
	}

	if err := crypto.ComparePassword(u.PasswordHash().String(), password.String()); err != nil {
		return nil, ErrInvalidPassword("password").Wrap(err)
	}

	return u, nil
}

func (s *Service) parseAndAuthenticate(
	ctx context.Context,
	rawUserID uuid.UUID,
	passwordRaw string,
) (fields.ID, *User, error) {
	id, err := fields.ParseRequiredID("id", rawUserID)
	if err != nil {
		return fields.ID{}, nil, err
	}

	password, err := ParsePassword("password", passwordRaw)
	if err != nil {
		return fields.ID{}, nil, err
	}

	u, err := s.fetchAndAuthenticate(ctx, id, password)
	if err != nil {
		return fields.ID{}, nil, err
	}

	return id, u, nil
}
