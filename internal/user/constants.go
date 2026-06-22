package user

// General
const (
	User = "user"
)

// Success messages
const (
	PingOK              = "Ping OK."
	CountOK             = "Count OK."
	CheckAvailabilityOK = "Check availability OK."
	CreateOK            = "Create OK."
	ListOK              = "List OK."
	ListUnverifiedOK    = "List Unverified OK."
	GetByIDOK           = "Get by ID OK."
	GetByEmailOK        = "Get by Email OK."
	GetByUsernameOK     = "Get by Username OK."
)

// Error messages
const (
	ErrInvalidCursor = "Invalid cursor; valid UUIDv7 format required."
	ErrInvalidID     = "Invalid ID."
)
