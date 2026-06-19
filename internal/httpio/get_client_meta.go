package httpio

import (
	"net/http"
	"net/netip"
)

type ClientMeta struct {
	IP        netip.Addr
	UserAgent string
	// Add future fields here, like:
	// DeviceID  string
	// Region    string
}

func GetClientMeta(r *http.Request) ClientMeta {
	return ClientMeta{
		IP:        GetClientIP(r, false),
		UserAgent: r.UserAgent(),
	}
}
