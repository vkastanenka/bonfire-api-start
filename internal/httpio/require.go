package httpio

import (
	"context"
	"net/http"
	"strings"

	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/token"
)

// RequireAuth returns a middleware that validates a Bearer access token and injects claims into the request context.
func RequireAuth(t *token.Provider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondError(w, r,
					errs.Unauthenticated("Missing authorization header.").
						ErrorInfoReason("AUTH_HEADER_MISSING"),
				)
				return
			}

			bearerPrefix := "bearer "
			if len(authHeader) < len(bearerPrefix) || !strings.EqualFold(authHeader[:len(bearerPrefix)], bearerPrefix) {
				respondError(w, r,
					errs.Unauthenticated("Invalid authorization header format.").
						ErrorInfoReason("AUTH_HEADER_INVALID"),
				)
				return
			}

			tokenStr := strings.TrimSpace(authHeader[len(bearerPrefix):])
			if tokenStr == "" {
				respondError(w, r,
					errs.Unauthenticated("Invalid authorization header format.").
						ErrorInfoReason("AUTH_HEADER_INVALID"),
				)
				return
			}

			claims, err := t.VerifyAccess(tokenStr)
			if err != nil {
				respondError(w, r, err)
				return
			}

			ctx := context.WithValue(r.Context(), ctxKeyClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
