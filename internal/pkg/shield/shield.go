package shield

import (
	"time"
)

const (
	DefaultMaxAttempts = 5
	DefaultFailuresTTL = 1 * time.Minute
	DefaultCooldownTTL = 1 * time.Minute
)

func cacheIdentifier(identifier string) string {
	return "{" + identifier + "}"
}

func FailuresKey(domain, action, identifier string) string {
	return "shield:failures:" + domain + ":" + action + ":" + cacheIdentifier(identifier)
}

func CooldownKey(domain, action, identifier string) string {
	return "shield:cooldown:" + domain + ":" + action + ":" + cacheIdentifier(identifier)
}
