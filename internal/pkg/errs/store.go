package errs

import "sync/atomic"

var domain atomic.Value

const defaultStoreDomain = "api.bonfire.app"

func init() {
	domain.Store(defaultStoreDomain)
}

func NewStore(appDomain string) {
	domain.Store(appDomain)
}

func getDomain() string {
	if v := domain.Load(); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultStoreDomain
}
