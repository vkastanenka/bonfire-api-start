package user

// General
const (
	Domain              = "user"
	DomainDeleteRequest = "user_delete_request"
	DomainProfile       = "user_profile"
)

// Success messages
const (
	// users
	PingOK              = "Ping OK."
	CountOK             = "Count OK."
	CheckAvailabilityOK = "Check availability OK."
	CreateOK            = "Create OK."
	ListOK              = "List OK."
	ListUnverifiedOK    = "List Unverified OK."
	GetByIDOK           = "Get by ID OK."
	GetByEmailOK        = "Get by Email OK."
	GetByUsernameOK     = "Get by Username OK."

	// user_delete_requests
	ListDeleteRequestsDueOK    = "List delete requests due OK."
	GetDeleteRequestByUserIDOK = "Get delete request by user ID OK."

	// user_profiles
	GetProfileByUserIDOK       = "Get profile by user ID OK."
	UpdateProfileDisplayNameOK = "Update profile display name OK."
)

// Error messages
const (
	ErrInvalidCursor = "Invalid cursor; valid UUIDv7 format required."
	ErrInvalidID     = "Invalid ID."
)
