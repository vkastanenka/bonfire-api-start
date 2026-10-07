package errs

import (
	"sync/atomic"
)

// domain holds the active global error domain string.
var domain atomic.Value

const defaultDomain = "api.domain.com"

func init() {
	domain.Store(defaultDomain)
}

// SetDomain sets the app domain name used when auto-populating domain metadata in ErrorInfo payloads.
func SetDomain(d string) {
	domain.Store(d)
}

// getDomain Returns the currently configured domain string.
// Returns defaultDomain if the stored value is empty.
func getDomain() string {
	if v, ok := domain.Load().(string); ok && v != "" {
		return v
	}
	return defaultDomain
}
