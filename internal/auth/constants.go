package auth

// Success messages
const (
	RegisterOkMsg     = "User register ok! 🥳🎉"
	LoginOkMsg        = "User login ok."
	RefreshTokenOkMsg = "Tokens rotated ok!"
)

// Validation / Error messages
const (
	ErrEmailTaken          = "This email address is already registered."
	ErrUsernameTaken       = "This username is already taken."
	ErrRegFailed           = "Registration failed due to unavailable credentials."
	ErrPasswordHashing     = "An unexpected error occurred while securing your account password."
	ErrMissingRefreshToken = "Missing refresh token, please log in."
	ErrCredentialsInvalid  = "Invalid credentials."
	ErrSessionInvalid      = "Invalid or expired session."
	ErrSessionMalformed    = "Malformed session payload."
	ErrSessionNotFound     = "Session not found."
	ErrSessionBlocked      = "Access denied. Session is blocked."
	ErrSessionExpired      = "Session expired. Please log in again."
)
