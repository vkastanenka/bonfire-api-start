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
	ErrMissingAuthHeader = "Missing authorization header."
	ErrInvalidAuthHeader = "Invalid authorization header format."
	ErrInvalidToken      = "Invalid or expired access token."
	ErrMissingAuthCtx    = "Missing authentication context."
	ErrUnverifiedEmail   = "Unverified email. Please complete verification via your registration email."
)

// --- MIDDLEWARE FUNCTIONS ---

// RequireAuth
func RequireAuth(tokenSvc *token.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				httpio.RespondError(w, r, apperr.New(apperr.CodeUnauthenticated, ErrMissingAuthHeader))
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" || parts[1] == "" {
				httpio.RespondError(w, r, apperr.New(apperr.CodeUnauthorized, ErrInvalidAuthHeader))
				return
			}

			claims, err := tokenSvc.VerifyAccess(parts[1])
			if err != nil {
				httpio.RespondError(w, r, apperr.New(apperr.CodeUnauthenticated, ErrInvalidToken))
				return
			}

			// Inject into context
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireVerified
func RequireVerified() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := GetClaimsFromContext(r.Context())
			if err != nil {
				httpio.RespondError(w, r, apperr.New(apperr.CodeInternal, ErrMissingAuthCtx))
				return
			}

			if !claims.IsVerified {
				httpio.RespondError(w, r, apperr.New(apperr.CodeUnauthenticated, ErrUnverifiedEmail))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserIDFromContext
func GetClaimsFromContext(ctx context.Context) (*token.Claims, error) {
	claims, ok := ctx.Value(claimsKey).(*token.Claims)
	if !ok {
		return nil, errors.New(ErrMissingAuthCtx)
	}
	return claims, nil
}
