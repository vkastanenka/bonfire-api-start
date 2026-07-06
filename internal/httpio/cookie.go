package httpio

import (
	"bonfire-api/internal/token"
	"net/http"
	"time"
)

const (
	CookieRefreshToken = "refresh_token"
)

func SetCookieRefreshToken(w http.ResponseWriter, tokenString string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieRefreshToken,
		Value:    tokenString,
		Path:     "/",
		Expires:  time.Now().Add(token.RefreshTokenTTL),
		MaxAge:   int(token.RefreshTokenTTL.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

func ClearCookieRefreshToken(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieRefreshToken,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}
