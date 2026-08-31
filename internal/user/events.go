package user

const (
	EventUpdatePresence = "user.update-presence"
	//
	EventUpdateEmail             = "user.update-email"
	EventUpdateUsername          = "user.update-username"
	EventUpdatePassword          = "user.update-password"
	EventUpdateProfile           = "user.update-profile"
	EventUpdatePreferredPresence = "user.update-preferred-presence"
	EventDisable                 = "user.disable"
	EventScheduleDelete          = "user.schedule-delete"
	EventAnonymized              = "user.anonymized"
)

type EventUpdatePresencePayload struct {
	UserID   string `json:"user_id"`
	Presence string `json:"presence"`
}

//

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
