package errs

import (
	"sync/atomic"
)

var domain atomic.Value

const defaultDomain = "api.domain.com"

func init() {
	domain.Store(defaultDomain)
}

func SetDomain(d string) {
	domain.Store(d)
}

func getDomain() string {
	if v, ok := domain.Load().(string); ok && v != "" {
		return v
	}
	return defaultDomain
}
