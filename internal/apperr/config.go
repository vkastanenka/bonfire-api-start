package apperr

import "sync/atomic"

var defaultDomain atomic.Value

func init() {
	defaultDomain.Store("unknown.service")
}

func Init(domain string) {
	if domain != "" {
		defaultDomain.Store(domain)
	}
}

func getDefaultDomain() string {
	return defaultDomain.Load().(string)
}
