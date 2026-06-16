package httpio

import (
	"net/http"
	"time"
)

const RefreshTokenCookieName = "refresh_token"

// SetRefreshTokenCookie centralizes secure cookie management for token issuance and rotation.
func SetRefreshTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour), // Matches backend service TTL
		HttpOnly: true,
		Secure:   true, // Requires HTTPS in production environments
		SameSite: http.SameSiteStrictMode,
	})
}

// ClearRefreshTokenCookie invalidates the client cookie immediately upon manual logout.
func ClearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0), // Explicitly sets date to Jan 1, 1970 to clear the record
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}
