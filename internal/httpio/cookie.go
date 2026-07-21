package httpio

import (
	"net/http"
	"time"
)

const CookieRefreshToken = "refresh_token"

func GetCookieRefreshToken(r *http.Request) (string, error) {
	cookie, err := r.Cookie(CookieRefreshToken)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

type SetCookieRefreshTokenParams struct {
	Token   string
	Expires time.Time
}

func SetCookieRefreshToken(w http.ResponseWriter, p SetCookieRefreshTokenParams) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieRefreshToken,
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
		Name:     CookieRefreshToken,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}
