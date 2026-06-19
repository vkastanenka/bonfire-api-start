package httpio

import (
	"net"
	"net/http" // if needed, but netip is standard
	"net/netip"
	"strings"
)

// GetClientIP extracts and parses the real client IP address.
// If parsing fails or no IP is found, it returns an invalid netip.Addr{}.
func GetClientIP(r *http.Request, trustProxy bool) netip.Addr {
	var rawIP string

	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if len(parts) > 0 {
				rawIP = strings.TrimSpace(parts[0])
			}
		}

		if rawIP == "" {
			if xri := r.Header.Get("X-Real-IP"); xri != "" {
				rawIP = strings.TrimSpace(xri)
			}
		}
	}

	if rawIP == "" {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			rawIP = r.RemoteAddr
		} else {
			rawIP = ip
		}
	}

	// Parse the string directly into netip.Addr at the HTTP boundary
	addr, err := netip.ParseAddr(rawIP)
	if err != nil {
		// Fallback/Safety: Return an unassigned or loopback address
		// so your app doesn't panic on corrupt client headers.
		return netip.IPv4Unspecified()
	}

	return addr
}
