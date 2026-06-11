package auth

import (
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/token"
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// contextKey is a custom type used to prevent context key collisions.
// If we just used a standard string like "userID", another package might
// accidentally overwrite it.
type contextKey string

const UserIDKey contextKey = "user_id"

// RequireAuth is a standard HTTP middleware that validates the Access Token.
func RequireAuth(accessSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Extract the Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				httpio.RespondJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "Missing authorization header. Please log in.",
				})
				return
			}

			// 2. Validate the Bearer prefix format (e.g., "Bearer eyJhbG...")
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				httpio.RespondJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "Invalid authorization header format. Expected 'Bearer <token>'.",
				})
				return
			}
			accessToken := parts[1]

			// 3. Cryptographically verify the token using your existing utility
			claims, err := token.VerifyJWT(accessToken, accessSecret)
			if err != nil {
				httpio.RespondJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "Invalid or expired access token.",
				})
				return
			}

			// 4. Inject the UserID into the request context
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			reqWithCtx := r.WithContext(ctx)

			// 5. Pass control to the next handler with the enriched context
			next.ServeHTTP(w, reqWithCtx)
		})
	}
}

// GetUserIDFromContext safely extracts the user ID from the request context.
// Use this inside your authenticated handlers.
func GetUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
	if !ok {
		// This should theoretically never happen if the middleware is correctly applied,
		// but it's crucial for avoiding runtime panics.
		return uuid.Nil, errors.New("user ID not found in context")
	}
	return userID, nil
}
