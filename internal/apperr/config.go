package apperr

import "sync/atomic"

var domain atomic.Value

func init() {
	domain.Store("unknown.service")
}

func Init(serviceDomain string) {
	if serviceDomain != "" {
		domain.Store(serviceDomain)
	}
}

func getDomain() string {
	return domain.Load().(string)
}
