package errs

import (
	"sync/atomic"
)

// domain holds the active global error domain string in atomic storage for thread-safe concurrent access.
var domain atomic.Value

// defaultDomain is the fallback domain string used when no custom domain is explicitly configured.
const defaultDomain = "api.domain.com"

func init() {
	domain.Store(defaultDomain)
}

// SetDomain sets the package-level domain name used when auto-populating domain metadata
// in ErrorInfo payloads. It is thread-safe for concurrent execution.
func SetDomain(d string) {
	domain.Store(d)
}

// getDomain is an internal unexported helper returning the currently configured domain string.
// Returns defaultDomain if the stored atomic value is empty or uninitialized.
func getDomain() string {
	if v, ok := domain.Load().(string); ok && v != "" {
		return v
	}
	return defaultDomain
}
