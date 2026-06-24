package httpio

import (
	"bonfire-api/internal/token"
	"net/http"
	"time"
)

// --- COOKIE CONSTANTS ---

const (
	RefreshTokenCookie = "refresh_token"
)

// --- COOKIE FUNCTIONS ---

// SetRefreshTokenCookie
func SetRefreshTokenCookie(w http.ResponseWriter, tokenString string) {
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    tokenString,
		Path:     "/",
		Expires:  time.Now().Add(token.RefreshTokenTTL),
		MaxAge:   int(token.RefreshTokenTTL.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

// ClearRefreshTokenCookie
func ClearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}
