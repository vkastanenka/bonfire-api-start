package httpio

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

type ClientMeta struct {
	IP        netip.Addr
	UserAgent string
	OS        string
	Browser   string
}

// WithClientMeta populates request context with IP, UserAgent, OS, and Browser details.
func WithClientMeta(trustProxy bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ua := r.UserAgent()
			os, browser := parseUserAgent(ua)

			meta := ClientMeta{
				IP:        extractIP(r, trustProxy),
				UserAgent: ua,
				OS:        os,
				Browser:   browser,
			}

			ctx := context.WithValue(r.Context(), ctxMetaKey, meta)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func parseUserAgent(ua string) (os string, browser string) {
	if ua == "" {
		return "Unknown", "Unknown"
	}

	uaLower := strings.ToLower(ua)

	switch {
	case strings.Contains(uaLower, "iphone") || strings.Contains(uaLower, "ipad") || strings.Contains(uaLower, "ipod"):
		os = "iOS"
	case strings.Contains(uaLower, "android"):
		os = "Android"
	case strings.Contains(uaLower, "windows"):
		os = "Windows"
	case strings.Contains(uaLower, "macintosh") || strings.Contains(uaLower, "mac os x"):
		os = "macOS"
	case strings.Contains(uaLower, "linux"):
		os = "Linux"
	default:
		os = "Unknown"
	}

	switch {
	case strings.Contains(uaLower, "edg/"):
		browser = "Edge"
	case strings.Contains(uaLower, "firefox") || strings.Contains(uaLower, "fxios"):
		browser = "Firefox"
	case strings.Contains(uaLower, "chrome") || strings.Contains(uaLower, "crios"):
		browser = "Chrome"
	case strings.Contains(uaLower, "safari"):
		browser = "Safari"
	default:
		browser = "Unknown"
	}

	return os, browser
}

func extractIP(r *http.Request, trustProxy bool) netip.Addr {
	if trustProxy {
		if apiIP := r.Header.Get("CF-Connecting-IP"); apiIP != "" {
			if addr, err := netip.ParseAddr(strings.TrimSpace(apiIP)); err == nil {
				return addr
			}
		}
		if apiIP := r.Header.Get("X-Real-IP"); apiIP != "" {
			if addr, err := netip.ParseAddr(strings.TrimSpace(apiIP)); err == nil {
				return addr
			}
		}
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if len(parts) > 0 {
				targetIP := strings.TrimSpace(parts[0])
				if addr, err := netip.ParseAddr(targetIP); err == nil {
					return addr
				}
			}
		}
	}

	rawIP := r.RemoteAddr
	if ip, _, err := net.SplitHostPort(rawIP); err == nil {
		rawIP = ip
	}

	rawIP = strings.TrimSuffix(strings.TrimPrefix(rawIP, "["), "]")

	addr, err := netip.ParseAddr(rawIP)
	if err != nil {
		return netip.IPv4Unspecified()
	}

	return addr
}
