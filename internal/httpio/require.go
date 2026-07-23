package httpio

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/token"
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
				respondError(w, r,
					errs.Unauthenticated(errMissingAuthHeader).
						Reason("AUTH_HEADER_MISSING"),
				)
				return
			}

			if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				respondError(w, r,
					errs.Unauthenticated(errInvalidAuthHeader).
						Reason("AUTH_HEADER_INVALID"),
				)
				return
			}

			tokenStr := strings.TrimSpace(authHeader[7:])
			if tokenStr == "" {
				respondError(w, r,
					errs.Unauthenticated(errInvalidAuthHeader).
						Reason("AUTH_HEADER_INVALID"),
				)
				return
			}

			claims, err := t.VerifyAccess(tokenStr)
			if err != nil {
				if errors.Is(err, token.ErrTokenExpired) {
					respondError(w, r,
						errs.Unauthenticated(errInvalidToken).
							Reason("TOKEN_EXPIRED").
							Wrap(err),
					)
					return
				}

				respondError(w, r,
					errs.Unauthenticated("Invalid or corrupt authorization token.").
						Reason("TOKEN_INVALID").
						Wrap(err),
				)
				return
			}

			ctx := context.WithValue(r.Context(), CtxClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
