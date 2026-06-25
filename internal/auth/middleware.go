package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/token"
	"context"
	"errors"
	"net/http"
	"strings"
)

// --- MIDDLEWARE TYPES ---

type contextKey string

// --- MIDDLEWARE CONSTANTS ---

const claimsKey contextKey = "user_claims"

const (
	errMissingAuthHeader = "Missing authorization header."
	errInvalidAuthHeader = "Invalid authorization header format."
	errInvalidToken      = "Invalid or expired access token."
	errMissingAuthCtx    = "Missing authentication context."
	errUnverifiedEmail   = "Unverified email. Please complete verification via your registration email."
)

// --- MIDDLEWARE FUNCTIONS ---

// RequireAuth validates the presence and validity of the Bearer access token.
func RequireAuth(tokenSvc *token.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				httpio.RespondError(w, r, apperr.New(apperr.CodeUnauthorized, errMissingAuthHeader))
				return
			}

			// Resilient prefix check
			if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				httpio.RespondError(w, r, apperr.New(apperr.CodeUnauthorized, errInvalidAuthHeader)) // Patched to CodeUnauthorized
				return
			}

			tokenStr := strings.TrimSpace(authHeader[7:])
			if tokenStr == "" {
				httpio.RespondError(w, r, apperr.New(apperr.CodeUnauthorized, errInvalidAuthHeader))
				return
			}

			claims, err := tokenSvc.VerifyAccess(tokenStr)
			if err != nil {
				httpio.RespondError(w, r, apperr.New(apperr.CodeUnauthorized, errInvalidToken))
				return
			}

			// Inject into context
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireVerified blocks authenticated requests if the email has not been confirmed.
func RequireVerified() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := GetClaimsFromContext(r.Context())
			if err != nil {
				httpio.RespondError(w, r, apperr.New(apperr.CodeInternal, errMissingAuthCtx))
				return
			}

			// Patched to CodeForbidden because we know identity, but refuse entry
			if !claims.IsVerified {
				httpio.RespondError(w, r, apperr.New(apperr.CodeForbidden, errUnverifiedEmail))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetClaimsFromContext extracts token claims from the request context.
func GetClaimsFromContext(ctx context.Context) (*token.Claims, error) {
	claims, ok := ctx.Value(claimsKey).(*token.Claims)
	if !ok {
		return nil, errors.New(errMissingAuthCtx)
	}
	return claims, nil
}
