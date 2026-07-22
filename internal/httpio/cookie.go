package httpio

import (
	"net/http"
	"time"
)

const CookieRefreshToken = "refresh_token"

func CookieGetRefreshToken(r *http.Request) (string, error) {
	cookie, err := r.Cookie(CookieRefreshToken)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

func CookieSetRefreshToken(w http.ResponseWriter, token string, expires time.Time) {
	maxAge := int(time.Until(expires).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}

	http.SetCookie(w, &http.Cookie{
		Name:     CookieRefreshToken,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

func CookieClearRefreshToken(w http.ResponseWriter) {
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
