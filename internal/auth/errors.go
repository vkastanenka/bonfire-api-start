package auth

import (
	"bonfire-api/internal/apperr"
)

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