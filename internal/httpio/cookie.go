package httpio

import (
	"net/http"
	"time"

	"bonfire-api/internal/token"
)

const cookieNameRefreshToken = "refresh_token"

func SetCookieRefreshToken(w http.ResponseWriter, tokenString string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieNameRefreshToken,
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
		Name:     cookieNameRefreshToken,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}
