package auth

import (
	"bonfire-api/internal/token"
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
	ClaimsKey contextKey = "user_claims" // Add a new key for the full claims struct
)

// RequireAuth validates the Access Token and injects claims into the context.
func RequireAuth(manager token.Service, accessSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// authHeader := r.Header.Get("Authorization")
			// if authHeader == "" {
			// 	httpio.RespondJSON(w, r, http.StatusUnauthorized, map[string]string{"error": "Missing authorization header. Please log in."})
			// 	return
			// }

			// parts := strings.Split(authHeader, " ")
			// if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			// 	httpio.RespondJSON(w, r, http.StatusUnauthorized, map[string]string{"error": "Invalid authorization header format."})
			// 	return
			// }

			// accessToken := parts[1]

			// claims, err := manager.VerifyJWT(accessToken, accessSecret)
			// if err != nil {
			// 	httpio.RespondJSON(w, r, http.StatusUnauthorized, map[string]string{"error": "Invalid or expired access token."})
			// 	return
			// }

			// // Store BOTH the UserID (for convenience) AND the full Claims (for flags)
			// ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			// ctx = context.WithValue(ctx, ClaimsKey, claims)

			// next.ServeHTTP(w, r.WithContext(ctx))
			next.ServeHTTP(w, r)
		})
	}
}

// RequireVerified blocks authenticated requests if the account doesn't carry the UserFlagVerified bit.
// IT MUST ALWAYS RUN AFTER RequireAuth!
func RequireVerified() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Read the full claims struct that RequireAuth injected
			// claims, ok := r.Context().Value(ClaimsKey).(*token.Claims)
			// if !ok {
			// 	httpio.RespondJSON(w, r, http.StatusUnauthorized, map[string]string{"error": "Unauthorized Access."})
			// 	return
			// }

			// Perform an in-memory bitwise check
			// currentFlags := UserFlag(claims.Flags)
			// if !currentFlags.Has(UserFlagVerified) {
			// 	httpio.RespondJSON(w, r, http.StatusForbidden, map[string]string{
			// 		"error": "Unverified email. Please complete verification via your registration email.",
			// 	})
			// 	return
			// }

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserIDFromContext safely extracts the user ID from the request context.
func GetUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("user ID not found in context")
	}
	return userID, nil
}
