package user

const (
	EventUpdatePresence = "user.update-presence"
	EventUpdateUsername = "user.update-username"
	EventUpdateProfile  = "user.update-profile"
	EventDisable        = "user.disable"
)

type EventUpdatePresencePayload struct {
	UserID   string `json:"user_id"`
	Presence string `json:"presence"`
}

type EventUpdateUsernamePayload struct {
	UserID      string `json:"user_id"`
	NewUsername string `json:"new_username"`
	UpdatedAt   string `json:"updated_at"`
}

type EventUpdateProfilePayload struct {
	UserID      string  `json:"user_id"`
	DisplayName string  `json:"display_name"`
	Bio         *string `json:"bio,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	BannerColor *string `json:"banner_color,omitempty"`
	UpdatedAt   string  `json:"updated_at"`
}

type EventUpdatePreferredPresencePayload struct {
	UserID            string  `json:"user_id"`
	PreferredPresence *string `json:"preferred_presence,omitempty"`
	Until             *string `json:"until,omitempty"`
}

type EventDisablePayload struct {
	UserID    string `json:"user_id"`
	UpdatedAt string `json:"updated_at"`
}
