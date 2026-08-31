package user

import (
	"bonfire-api/internal/fields"
	"context"
	"log/slog"
)

const (
	WSUpdatePresence = "user.update-presence"
)

type WSUpdatePresenceData struct {
	UserID   string `json:"user_id"`
	Presence string `json:"presence"`
}

type Events struct {
	cache Cache
	repo  Repository
}

func NewEvents(cache Cache, repo Repository) *Events {
	return &Events{cache: cache, repo: repo}
}

// UpdatePresence sets a user's presence in the cache and broadcasts it.
func (e *Events) UpdatePresence(ctx context.Context, actorUserID fields.ID, payload WSUpdatePresenceData) error {
	if actorUserID.String() != payload.UserID {
		return nil
	}

	p, err := ParsePresenceString(payload.Presence)
	if err != nil {
		slog.WarnContext(ctx, "Invalid presence string format, defaulting to online",
			"user_id", actorUserID,
			"presence", payload.Presence,
			"error", err,
		)
		if err := e.cache.SetPresence(ctx, actorUserID, NewPresenceOnline()); err != nil {
			slog.ErrorContext(ctx, "Failed to update user presence cache",
				"user_id", actorUserID,
				"error", err,
			)
		}
	}

	if err := e.cache.SetPresence(ctx, actorUserID, p); err != nil {
		slog.ErrorContext(ctx, "Failed to update user presence cache",
			"user_id", actorUserID,
			"error", err,
		)
	}

	// // 3. FETCH RECIPIENTS: Get friends + channel peers from Redis sets (0 DB hits)
	// recipientIDs, err := e.cache.GetPresenceRecipients(ctx, actorUserID)
	// if err != nil {
	//     slog.ErrorContext(ctx, "Failed to fetch presence recipients", "user_id", actorUserID, "error", err)
	//     return nil // Drop broadcast gracefully on cache miss
	// }

	// if len(recipientIDs) == 0 {
	//     return nil
	// }

	// // 4. MAP TO NODES: Find which gateway node each recipient is currently connected to
	// // Returns map[nodeID][]userID
	// nodeUserMap, err := e.cache.GetNodeIDsForUsers(ctx, recipientIDs)
	// if err != nil || len(nodeUserMap) == 0 {
	//     return nil // All recipients are offline
	// }

	// // Prepare the outbound payload
	// outboundMsg := OutboundWSMessage{
	//     Type: "user.presence-updated",
	//     Data: map[string]any{
	//         "user_id":  actorUserID.String(),
	//         "presence": p.String(),
	//     },
	// }

	// // 5. PUBLISH: Send one message per unique Gateway Node holding recipient connections
	// for nodeID, targetUserIDs := range nodeUserMap {
	//     channelName := fmt.Sprintf("gateway:%s:events", nodeID)

	//     nodePayload := GatewayTargetPayload{
	//         TargetUserIDs: targetUserIDs,
	//         Message:       outboundMsg,
	//     }

	//     // Use your existing redis.Publish function here!
	//     if err := pkgredis.Publish(ctx, e.redisClient, channelName, nodePayload); err != nil {
	//         slog.ErrorContext(ctx, "Failed to publish presence update to gateway node",
	//             "node_id", nodeID,
	//             "error", err,
	//         )
	//     }
	// }

	return nil
}

// const (
// 	EventUpdateEmail             = "user.update-email"
// 	EventUpdateUsername          = "user.update-username"
// 	EventUpdatePassword          = "user.update-password"
// 	EventUpdateProfile           = "user.update-profile"
// 	EventUpdatePreferredPresence = "user.update-preferred-presence"
// 	EventDisable                 = "user.disable"
// 	EventScheduleDelete          = "user.schedule-delete"
// 	EventAnonymized              = "user.anonymized"
// )

// type EventUpdateEmailPayload struct {
// 	UserID   string `json:"user_id"`
// 	OldEmail string `json:"old_email"`
// 	NewEmail string `json:"new_email"`
// }

// type EventUpdateUsernamePayload struct {
// 	UserID      string `json:"user_id"`
// 	OldUsername string `json:"old_username"`
// 	NewUsername string `json:"new_username"`
// }

// type EventUpdatePasswordPayload struct {
// 	UserID string `json:"user_id"`
// 	Email  string `json:"email"`
// }

// type EventUpdateProfilePayload struct {
// 	UserID      string  `json:"user_id"`
// 	DisplayName string  `json:"display_name"`
// 	Bio         *string `json:"bio,omitempty"`
// 	AvatarURL   *string `json:"avatar_url,omitempty"`
// 	BannerColor *string `json:"banner_color,omitempty"`
// }

// type EventUpdatePreferredPresencePayload struct {
// 	UserID            string  `json:"user_id"`
// 	PreferredPresence *string `json:"preferred_presence,omitempty"`
// 	Until             *string `json:"until,omitempty"`
// }

// type EventDisablePayload struct {
// 	UserID string `json:"user_id"`
// }

// type EventScheduleDeletePayload struct {
// 	UserID      string `json:"user_id"`
// 	Email       string `json:"email"`
// 	ScheduledAt string `json:"scheduled_at"`
// }

// type EventAnonymizedPayload struct {
// 	UserID string `json:"user_id"`
// }

// // RegisterOutboxHandlers registers all user domain outbox processors.
// func RegisterOutboxHandlers(w *outbox.Worker, mailer email.Mailer, cacheStore *cache.Store) {
// 	pubToUser := func(ctx context.Context, userID string, eventType string, payload any) error {
// 		channel := fmt.Sprintf("user:%s:events", userID)
// 		wsEvent := map[string]any{
// 			"type": eventType,
// 			"data": payload,
// 		}
// 		return cacheStore.Publish(ctx, channel, wsEvent)
// 	}

// 	handleUserTeardown := func(ctx context.Context, userID string, eventType string) error {
// 		// Invalidate cache sessions
// 		sessionKey := fmt.Sprintf("user:%s:session", userID)
// 		_ = cacheStore.Delete(ctx, sessionKey) // Non-fatal, proceed to disconnect WS

// 		// Force gateway to sever open WS connection
// 		return pubToUser(ctx, userID, eventType, map[string]string{"user_id": userID})
// 	}

// 	// 1. Email & Password Security Alerts
// 	w.RegisterHandler(EventUpdateEmail, func(ctx context.Context, raw json.RawMessage) error {
// 		var p EventUpdateEmailPayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed update email payload: %v", outbox.ErrFatal, err)
// 		}
// 		return mailer.SendSecurityAlertEmail(ctx, p.OldEmail, "Your email address was updated.")
// 	})

// 	w.RegisterHandler(EventUpdatePassword, func(ctx context.Context, raw json.RawMessage) error {
// 		var p EventUpdatePasswordPayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed update password payload: %v", outbox.ErrFatal, err)
// 		}
// 		return mailer.SendSecurityAlertEmail(ctx, p.Email, "Your password was recently changed.")
// 	})

// 	// 2. Real-time User Updates over WebSocket
// 	w.RegisterHandler(EventUpdateProfile, func(ctx context.Context, raw json.RawMessage) error {
// 		var p EventUpdateProfilePayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed profile update payload: %v", outbox.ErrFatal, err)
// 		}
// 		return pubToUser(ctx, p.UserID, EventUpdateProfile, p)
// 	})

// 	w.RegisterHandler(EventUpdateUsername, func(ctx context.Context, raw json.RawMessage) error {
// 		var p EventUpdateUsernamePayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed username update payload: %v", outbox.ErrFatal, err)
// 		}
// 		return pubToUser(ctx, p.UserID, EventUpdateUsername, p)
// 	})

// 	w.RegisterHandler(EventUpdatePreferredPresence, func(ctx context.Context, raw json.RawMessage) error {
// 		var p EventUpdatePreferredPresencePayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed preferred presence update payload: %v", outbox.ErrFatal, err)
// 		}
// 		return pubToUser(ctx, p.UserID, EventUpdatePreferredPresence, p)
// 	})

// 	// 3. Lifecycle & Teardown Events
// 	w.RegisterHandler(EventScheduleDelete, func(ctx context.Context, raw json.RawMessage) error {
// 		var p EventScheduleDeletePayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed schedule delete payload: %v", outbox.ErrFatal, err)
// 		}

// 		msg := fmt.Sprintf("Your account has been scheduled for deletion on %s. If you did not request this, please log in to cancel.", p.ScheduledAt)
// 		if err := mailer.SendSecurityAlertEmail(ctx, p.Email, msg); err != nil {
// 			return err
// 		}

// 		return handleUserTeardown(ctx, p.UserID, EventScheduleDelete)
// 	})

// 	w.RegisterHandler(EventDisable, func(ctx context.Context, raw json.RawMessage) error {
// 		var p EventDisablePayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed disable payload: %v", outbox.ErrFatal, err)
// 		}
// 		return handleUserTeardown(ctx, p.UserID, EventDisable)
// 	})

// 	w.RegisterHandler(EventAnonymized, func(ctx context.Context, raw json.RawMessage) error {
// 		var p EventAnonymizedPayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed anonymized payload: %v", outbox.ErrFatal, err)
// 		}
// 		return handleUserTeardown(ctx, p.UserID, EventAnonymized)
// 	})
// }
