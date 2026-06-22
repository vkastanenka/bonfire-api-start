package outbox

// General
const (
	OutboxEvent = "outbox event"
)

// Success messages
const (
	PingOK           = "Ping OK."
	CountOK          = "Count OK."
	ListOK           = "List OK."
	GetByIDOK        = "Get by ID OK."
	ResetAttemptsOK  = "Reset attempts OK."
	DeleteByIDOK     = "Delete by ID OK."
	PurgeProcessedOK = "Delete by ID OK."
)

// Error messages
const (
	ErrInvalidCursor = "Invalid cursor; valid UUIDv7 format required."
	ErrInvalidID     = "Invalid ID."
)
