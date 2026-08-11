package session

const (
	EventSessionRevoke    = "session.revoke"
	EventSessionRevokeAll = "session.revoke-all"
)

type EventSessionRevokePayload struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	RevokedAt string `json:"revoked_at"`
}

type EventSessionRevokeAllPayload struct {
	UserID     string   `json:"user_id"`
	SessionIDs []string `json:"session_ids"`
	RevokedAt  string   `json:"revoked_at"`
}
