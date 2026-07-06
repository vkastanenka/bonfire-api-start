package httpio

import (
	"net/http"
	"time"
)

const cookieNameRefreshToken = "refresh_token"

type SetCookieRefreshTokenParams struct {
	Token   string
	Expires time.Time
}

func SetCookieRefreshToken(w http.ResponseWriter, p SetCookieRefreshTokenParams) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieNameRefreshToken,
		Value:    p.Token,
		Path:     "/",
		Expires:  p.Expires,
		MaxAge:   int(time.Until(p.Expires).Seconds()),
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
