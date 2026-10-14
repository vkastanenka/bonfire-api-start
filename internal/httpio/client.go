package httpio

import (
	"bonfire-api/internal/appctx"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/mileusna/useragent"
)

// WithClientMeta extracts IP, UserAgent, OS, and Browser details and attaches them to the request context.
func WithClientMeta(trustProxy bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r, trustProxy)
			ua := r.UserAgent()
			os, client := parseOSClient(ua)

			meta := appctx.ClientMeta{
				IP:        ip,
				UserAgent: ua,
				OS:        os,
				Client:    client,
			}

			ctx := appctx.WithClientMeta(r.Context(), meta)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// parseOSClient performs OS and client detection from a UserAgent string.
func parseOSClient(ua string) (os string, client string) {
	if ua == "" {
		return "UNKNOWN", "UNKNOWN"
	}

	result := useragent.Parse(ua)

	os = result.OS
	if os == "" {
		os = "UNKNOWN"
	}

	client = result.Name
	if client == "" {
		client = "UNKNOWN"
	}

	return os, client
}

// extractIP resolves the client IP address from proxy headers or remote connection address.
func extractIP(r *http.Request, trustProxy bool) netip.Addr {
	if trustProxy {
		for _, header := range []string{"CF-Connecting-IP", "X-Real-IP"} {
			if val := strings.TrimSpace(r.Header.Get(header)); val != "" {
				if addr, err := netip.ParseAddr(val); err == nil {
					return addr
				}
			}
		}

		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if parts := strings.Split(xff, ","); len(parts) > 0 {
				if addr, err := netip.ParseAddr(strings.TrimSpace(parts[0])); err == nil {
					return addr
				}
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	if addr, err := netip.ParseAddr(strings.Trim(host, "[]")); err == nil {
		return addr
	}

	return netip.IPv4Unspecified()
}
