package user

import (
	"bonfire-api/internal/email"
	"bonfire-api/internal/outbox"
	"context"
	"encoding/json"
	"fmt"
)

const (
	EventUpdateEmail             = "user.update-email"
	EventUpdateUsername          = "user.update-username"
	EventUpdatePassword          = "user.update-password"
	EventUpdateProfile           = "user.update-profile"
	EventUpdatePreferredPresence = "user.update-preferred-presence"
	EventDisable                 = "user.disable"
	EventAnonymized              = "user.anonymized"
)

type EventUpdateEmailPayload struct {
	UserID   string `json:"user_id"`
	OldEmail string `json:"old_email"`
	NewEmail string `json:"new_email"`
}

type EventUpdateUsernamePayload struct {
	UserID      string `json:"user_id"`
	OldUsername string `json:"old_username"`
	NewUsername string `json:"new_username"`
}

type EventUpdatePasswordPayload struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

type EventUpdateProfilePayload struct {
	UserID      string  `json:"user_id"`
	DisplayName string  `json:"display_name"`
	Bio         *string `json:"bio,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	BannerColor *string `json:"banner_color,omitempty"`
}

type EventUpdatePreferredPresencePayload struct {
	UserID            string  `json:"user_id"`
	PreferredPresence *string `json:"preferred_presence,omitempty"`
	Until             *string `json:"until,omitempty"`
}

type EventDisablePayload struct {
	UserID string `json:"user_id"`
}

type EventAnonymizedPayload struct {
	UserID string `json:"user_id"`
}

// RegisterOutboxHandlers registers all user domain outbox processors.
func RegisterOutboxHandlers(w *outbox.Worker, mailer email.Mailer, cacheStore cache.Store) {
	// 1. Send Security Notification on Email Change
	w.RegisterHandler(EventUpdateEmail, func(ctx context.Context, raw json.RawMessage) error {
		var p EventUpdateEmailPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed update email payload: %v", outbox.ErrFatal, err)
		}
		// Notify old email address about the update
		return mailer.SendSecurityAlertEmail(ctx, p.OldEmail, "Your email address was updated.")
	})

	// 2. Send Security Notification on Password Change
	w.RegisterHandler(EventUpdatePassword, func(ctx context.Context, raw json.RawMessage) error {
		var p EventUpdatePasswordPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed update password payload: %v", outbox.ErrFatal, err)
		}
		return mailer.SendSecurityAlertEmail(ctx, p.Email, "Your password was recently changed.")
	})

	// 3. Broadcast Profile / Username Updates to Connected WebSocket Clients via Redis
	w.RegisterHandler(EventUpdateProfile, func(ctx context.Context, raw json.RawMessage) error {
		var p EventUpdateProfilePayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed profile update payload: %v", outbox.ErrFatal, err)
		}

		channel := fmt.Sprintf("user:%s:events", p.UserID)
		eventMessage := map[string]any{
			"type": EventUpdateProfile,
			"data": p,
		}

		data, err := json.Marshal(eventMessage)
		if err != nil {
			return fmt.Errorf("%w: failed to marshal websocket payload: %v", outbox.ErrFatal, err)
		}

		return cacheStore.Publish(ctx, channel, string(data))
	})

	w.RegisterHandler(EventUpdateUsername, func(ctx context.Context, raw json.RawMessage) error {
		var p EventUpdateUsernamePayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed username update payload: %v", outbox.ErrFatal, err)
		}

		channel := fmt.Sprintf("user:%s:events", p.UserID)
		eventMessage := map[string]any{
			"type": EventUpdateUsername,
			"data": p,
		}

		data, err := json.Marshal(eventMessage)
		if err != nil {
			return fmt.Errorf("%w: failed to marshal username payload: %v", outbox.ErrFatal, err)
		}

		return cacheStore.Publish(ctx, channel, string(data))
	})

	// 4. Teardown Active User Sessions on Disable or Anonymization
	handleUserTeardown := func(ctx context.Context, userID string, eventType string) error {
		// Invalidate cache sessions
		sessionKey := fmt.Sprintf("user:%s:session", userID)
		if err := cacheStore.Delete(ctx, sessionKey); err != nil {
			// Non-fatal, keep going to force WS disconnect
		}

		// Force gateway to sever open WS connection
		channel := fmt.Sprintf("user:%s:events", userID)
		eventMessage := map[string]any{
			"type": eventType,
			"data": map[string]string{"user_id": userID},
		}

		data, _ := json.Marshal(eventMessage)
		return cacheStore.Publish(ctx, channel, string(data))
	}

	w.RegisterHandler(EventDisable, func(ctx context.Context, raw json.RawMessage) error {
		var p EventDisablePayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed disable payload: %v", outbox.ErrFatal, err)
		}
		return handleUserTeardown(ctx, p.UserID, EventDisable)
	})

	w.RegisterHandler(EventAnonymized, func(ctx context.Context, raw json.RawMessage) error {
		var p EventAnonymizedPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed anonymized payload: %v", outbox.ErrFatal, err)
		}
		return handleUserTeardown(ctx, p.UserID, EventAnonymized)
	})
}
