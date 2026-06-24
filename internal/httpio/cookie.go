package httpio

import (
	"net/http"
	"time"
)

// --- COOKIE CONSTANTS ---

const (
	RefreshTokenCookie = "refresh_token"
	RefreshTokenTTL    = 7 * 24 * time.Hour
)

// --- COOKIE FUNCTIONS ---

// SetRefreshTokenCookie
func SetRefreshTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookie,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(RefreshTokenTTL),
		MaxAge:   int(RefreshTokenTTL.Seconds()),
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
