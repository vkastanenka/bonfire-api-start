package auth

// Success messages
const (
	RegisterOkMsg          = "User register ok! 🥳🎉"
	LoginOkMsg             = "User login ok!"
	RefreshTokenOkMsg      = "Tokens rotated successfully!"
	RefreshTokenMissingMsg = "Missing refresh token! Please log in!"
)

// Validation / Error messages
const (
	ErrEmailTaken         = "This email address is already registered."
	ErrUsernameTaken      = "This username is already taken."
	ErrRegFailed          = "Registration failed due to unavailable credentials."
	ErrPasswordHashing    = "An unexpected error occurred while securing your account password."
)

// Event types
const (
	EventUserRegistered = "user.registered"
)
