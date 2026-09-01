package user

const (
	EventUpdateUsername = "user.update-username"
	EventUpdatePresence = "user.update-presence"
	EventUpdateProfile  = "user.update-profile"
	EventDisable        = "user.disable"
)

type EventUpdateUsernamePayload struct {
	UserID    string `json:"user_id"`
	Username  string `json:"new_username"`
	UpdatedAt string `json:"updated_at"`
}

type EventUpdatePresencePayload struct {
	UserID    string `json:"user_id"`
	Presence  string `json:"presence"`
	UpdatedAt string `json:"updated_at"`
}

type EventUpdateProfilePayload struct {
	UserID      string  `json:"user_id"`
	DisplayName string  `json:"display_name"`
	Bio         *string `json:"bio,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	BannerColor *string `json:"banner_color,omitempty"`
	UpdatedAt   string  `json:"updated_at"`
}

type EventDisablePayload struct {
	UserID    string `json:"user_id"`
	UpdatedAt string `json:"updated_at"`
}
