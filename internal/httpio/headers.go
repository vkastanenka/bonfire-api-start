package httpio

import "net/http"

// SecurityHeaders sets standard web and API security headers on outgoing HTTP responses.
func SecurityHeaders(isProd bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent MIME-type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Prevent clickjacking by disallowing framing
			w.Header().Set("X-Frame-Options", "DENY")

			// Restrict referrer information sent in request headers
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Restrict client-side browser feature access
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

			// Modern CSP for API responses restricting resource loading and framing
			w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none';")

			// Force HTTPS connections in production environments
			if isProd {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}
