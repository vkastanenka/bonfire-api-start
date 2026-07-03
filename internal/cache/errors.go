package cache

import (
	"bonfire-api/internal/apperr"
	"context"
	"errors"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

// Domain represents an explicit database domain resource.
type Domain string

const (
	DomainAuth      Domain = "auth"
	DomainPresence  Domain = "presence"
	DomainStatus    Domain = "status"
	DomainEvents    Domain = "events"
	DomainRateLimit Domain = "rate_limit"
)

func (r Domain) String() string {
	return string(r)
}

// Errors.
var ErrCacheMiss = errors.New("cache: key not found")

// NewError converts native Go-Redis driver faults into domain apperr classifications.
func NewError(err error, resource Domain) error {
	if err == nil {
		return nil
	}

	// Don't intercept existing domain app errors
	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		return err
	}

	// Intercept Cache Misses as Not Found
	if IsCacheMissError(err) {
		return apperr.NewNotFound(err, fmt.Sprintf("%s was not found in cache.", resource))
	}

	// Intercept Redis Timeouts / Context Cancellations
	if errors.Is(err, context.DeadlineExceeded) {
		return apperr.NewRequestTimeout(err, "The cache operation timed out.")
	}
	if errors.Is(err, context.Canceled) {
		return apperr.NewInvalidInput(err, "The cache transaction was aborted by the client.")
	}

	// Default fallback for connection drops or broker failures
	return apperr.NewInternal(err, "A temporary caching error occurred.")
}

// IsCacheMissError checks if an error indicates a missing cache key.
func IsCacheMissError(err error) bool {
	return errors.Is(err, goredis.Nil) || errors.Is(err, ErrCacheMiss)
}
