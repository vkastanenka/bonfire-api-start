package httpio

import (
	"net/http"
	"os"
	"strings"
	"time"
)

const CookieRefreshToken = "refresh-token"

// CookieGetRefreshToken extracts the refresh token string from the request cookies.
func CookieGetRefreshToken(r *http.Request) (string, error) {
	cookie, err := r.Cookie(CookieRefreshToken)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

// CookieSetRefreshToken sets an HttpOnly, Secure refresh token cookie with environment-aware SameSite.
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
		SameSite: getSameSite(),
	})
}

// CookieClearRefreshToken invalidates the refresh token cookie by setting an expired MaxAge.
func CookieClearRefreshToken(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieRefreshToken,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: getSameSite(),
	})
}

// getSameSite dynamically resolves SameSite mode based on the environment.
func getSameSite() http.SameSite {
	env := strings.ToLower(os.Getenv("APP_ENV"))
	if env == "production" || env == "prod" {
		return http.SameSiteStrictMode
	}
	return http.SameSiteLaxMode
}
