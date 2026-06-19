package auth

// Success messages
const (
	RegisterOk     = "User register ok! 🥳🎉"
	LoginOk        = "User login ok!"
	RefreshTokenOk = "Refresh tokens ok!"
)

// Validation / Error messages
const (
	ErrEmailTaken          = "Email taken."
	ErrUsernameTaken       = "Username taken."
	ErrRegFailed           = "Registration failed."
	ErrHashPassword        = "Hash password failed."
	ErrMissingRefreshToken = "Missing refresh token, please log in."
	ErrCredentialsInvalid  = "Invalid credentials."
	ErrSessionInvalid      = "Invalid or expired session."
	ErrSessionMalformed    = "Malformed session payload."
	ErrSessionNotFound     = "Session not found."
	ErrSessionBlocked      = "Access denied. Session is blocked."
	ErrSessionExpired      = "Session expired. Please log in."
)
