package errs

import "sync/atomic"

var domain atomic.Value

func init() {
	domain.Store("unknown.service")
}

func NewStore(serviceDomain string) {
	if serviceDomain != "" {
		domain.Store(serviceDomain)
	}
}

func getDomain() string {
	if v := domain.Load(); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return "unknown.service"
}
