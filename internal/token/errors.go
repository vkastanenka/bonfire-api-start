package token

import "bonfire-api/internal/pkg/errs"

const (
	ErrReasonTokenConfigInvalid    = "TOKEN_CONFIG_INVALID"
	ErrReasonTokenConfigMissing    = "TOKEN_CONFIG_MISSING"
	ErrReasonTokenSignFailed       = "TOKEN_SIGN_FAILED"
	ErrReasonTokenExpired          = "TOKEN_EXPIRED"
	ErrReasonTokenIssuerInvalid    = "TOKEN_ISSUER_INVALID"
	ErrReasonTokenMalformed        = "TOKEN_MALFORMED"
	ErrReasonTokenSignatureInvalid = "TOKEN_SIGNATURE_INVALID"
	ErrReasonTokenInvalid          = "TOKEN_INVALID"
	ErrReasonTokenTypeInvalid      = "TOKEN_TYPE_INVALID"
	ErrReasonTokenClaimsInvalid    = "TOKEN_CLAIMS_INVALID"
)

// ErrTokenConfigInvalid returns an InvalidArgument error when token provider setup fails validation.
func ErrTokenConfigInvalid() *errs.Error {
	return errs.InvalidArgument("Invalid token configuration.").
		Reason(ErrReasonTokenConfigInvalid)
}

// ErrTokenConfigMissing returns an Internal error when provider state is missing expected variant config at runtime.
func ErrTokenConfigMissing() *errs.Error {
	return errs.Internal("Missing token configuration.").
		Reason(ErrReasonTokenConfigMissing)
}

// ErrTokenSignFailed returns an Internal error when signing a JWT string fails.
func ErrTokenSignFailed() *errs.Error {
	return errs.Internal("Failed to sign token.").
		Reason(ErrReasonTokenSignFailed)
}

// ErrTokenExpired returns an Unauthenticated error when the token's exp claim is in the past.
func ErrTokenExpired() *errs.Error {
	return errs.Unauthenticated("Token has expired.").
		Reason(ErrReasonTokenExpired)
}

// ErrTokenIssuerInvalid returns an Unauthenticated error when the iss claim does not match the provider.
func ErrTokenIssuerInvalid() *errs.Error {
	return errs.Unauthenticated("Invalid token issuer.").
		Reason(ErrReasonTokenIssuerInvalid)
}

// ErrTokenMalformed returns an InvalidArgument error when the raw JWT string structure is invalid.
func ErrTokenMalformed() *errs.Error {
	return errs.InvalidArgument("Token is malformed.").
		Reason(ErrReasonTokenMalformed).
		FieldViolation("token", "provided JWT string is malformed or unparseable", "INVALID_FORMAT")
}

// ErrTokenSignatureInvalid returns an Unauthenticated error when cryptographic signature validation fails.
func ErrTokenSignatureInvalid() *errs.Error {
	return errs.Unauthenticated("Invalid token signature.").
		Reason(ErrReasonTokenSignatureInvalid)
}

// ErrTokenInvalid returns a fallback Unauthenticated error when token parsing fails.
func ErrTokenInvalid() *errs.Error {
	return errs.Unauthenticated("Invalid token.").
		Reason(ErrReasonTokenInvalid)
}

// ErrTokenTypeInvalid returns an Unauthenticated error when a token variant does not match expectations.
func ErrTokenTypeInvalid() *errs.Error {
	return errs.Unauthenticated("Invalid token type.").
		Reason(ErrReasonTokenTypeInvalid)
}

// ErrTokenClaimsInvalid returns an Unauthenticated error when claim interface assertion or validity flags fail.
func ErrTokenClaimsInvalid() *errs.Error {
	return errs.Unauthenticated("Invalid token claims.").
		Reason(ErrReasonTokenClaimsInvalid)
}
