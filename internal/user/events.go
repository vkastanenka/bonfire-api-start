package user

const (
	EventUpdateEmail             = "user.update-email"
	EventUpdateUsername          = "user.update-username"
	EventUpdatePassword          = "user.update-password"
	EventUpdateProfile           = "user.update-profile"
	EventUpdatePreferredPresence = "user.update-preferred-presence"
	EventDisable                 = "user.disable"
	EventScheduleDelete          = "user.schedule-delete"
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

type EventScheduleDeletePayload struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	ScheduledAt string `json:"scheduled_at"`
}

type EventAnonymizedPayload struct {
	UserID string `json:"user_id"`
}

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
