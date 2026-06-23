package userprofile

// General
const (
	Domain = "user session"
)

// Success messages
const (
	PingOK               = "Ping OK."
	CountOK              = "Count OK."
	CreateOK             = "Create OK."
	GetByUserIDOK        = "Get by user ID OK."
	UpdateDisplayNameOK  = "Update display name OK."
	CountBytesOK         = "user_session:count:ok"
	ListActiveOK         = "user_session:list_active:ok"
	GetByIDOK            = "user_session:get_by_id:ok"
	GetByRefreshTokenOK  = "user_session:get_by_refresh_token:ok"
	UpdateRefreshTokenOK = "user_session:update_refresh_token:ok"
	UpdateLastSeenOK     = "user_session:update_last_seen:ok"
	MarkBlockedOK        = "user_session:mark_blocked:ok"
	PurgeExpiredOK       = "user_session:purge_expired:ok"
)

// Error messages
const (
	ErrInvalidID = "Invalid ID."
)
