package auth

import "bonfire-api/internal/apperr"

// General
const (
	DomainUserSession = "user session"
)

// Success messages
const (
	RegisterOK                     = "User register ok! 🥳🎉"
	LoginOK                        = "User login ok!"
	RefreshTokenOK                 = "Refresh tokens ok!"
	PingOK                         = "Ping ok!"
	CountUserSessionsOK            = "Count user sessions ok!"
	CreateUserSessionOK            = "Create user session ok!"
	ListActiveUserSessionByUserID  = "List active user session by user id ok!"
	GetUserSessionByIDOK           = "Get user session by id ok!"
	GetUserSessionByRefreshTokenOK = "Get user session by refresh token ok!"
	UpdateRefreshTokenOK           = "Update refresh token ok!"
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

// Errors

func NewLoginCredentialsError() error {
	return apperr.New(
		apperr.CodeUnauthenticated,
		ErrCredentialsInvalid,
		apperr.WithInvalidParams([]apperr.InvalidParam{
			{Name: "email", Reason: ErrCredentialsInvalid},
			{Name: "password", Reason: ErrCredentialsInvalid},
		}),
	)
}

func NewHashPasswordError(err error) error {
	return apperr.New(apperr.CodeInternal,
		ErrHashPassword,
		apperr.WithErr(err),
	)
}

func NewRegisterConflictError(emailAvailable, usernameAvailable bool) error {
	var params []apperr.InvalidParam

	if !emailAvailable {
		params = append(params, apperr.InvalidParam{Name: "email", Reason: ErrEmailTaken})
	}
	if !usernameAvailable {
		params = append(params, apperr.InvalidParam{Name: "username", Reason: ErrUsernameTaken})
	}

	return apperr.New(
		apperr.CodeConflict,
		ErrRegFailed,
		apperr.WithInvalidParams(params),
	)
}
