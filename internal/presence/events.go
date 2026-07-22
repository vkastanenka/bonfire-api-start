package presence

const (
	EventUpdated = "presence.updated"
)

type PresenceUpdatedPayload struct {
	UserID   string `json:"user_id"`
	Presence string `json:"presence"`
}
