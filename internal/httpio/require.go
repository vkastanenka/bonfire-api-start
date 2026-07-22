package httpio

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/token"
	"context"
	"errors"
	"net/http"
	"strings"
)

const (
	errMissingAuthHeader = "Missing authorization header."
	errInvalidAuthHeader = "Invalid authorization header format."
	errInvalidToken      = "Invalid or expired access token."
	errMissingAuthCtx    = "Missing authentication context."
	errUnverifiedEmail   = "Unverified email. Please complete verification via your registration email."
)

func RequireAuth(t *token.Provider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondError(w, r, apperr.NewUnauthenticated(nil, apperr.WithMsg(errMissingAuthHeader)))
				return
			}

			if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				respondError(w, r, apperr.NewUnauthenticated(nil, apperr.WithMsg(errInvalidAuthHeader)))
				return
			}

			tokenStr := strings.TrimSpace(authHeader[7:])
			if tokenStr == "" {
				respondError(w, r, apperr.NewUnauthenticated(nil, apperr.WithMsg(errInvalidAuthHeader)))
				return
			}

			claims, err := t.VerifyAccess(tokenStr)
			if err != nil {
				if errors.Is(err, token.ErrTokenExpired) {
					respondError(w, r,
						apperr.NewPermissionDenied(err, apperr.WithMsg(errInvalidToken)),
					)
					return
				}

				respondError(w, r, apperr.NewUnauthenticated(err, apperr.WithMsg("Invalid or corrupt authorization token.")))
				return
			}

			ctx := context.WithValue(r.Context(), ctxClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
