package redis

import (
	"bonfire-api/internal/apperr" // Assuming your apperr package path
	"context"
	"errors"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

// ErrCacheMiss standardizes the cache-miss state application-wide.
var ErrCacheMiss = errors.New("cache: key not found")

// IsCacheMissError checks if an error indicates a missing cache key.
func IsCacheMissError(err error) bool {
	return errors.Is(err, goredis.Nil) || errors.Is(err, ErrCacheMiss)
}

// NewError converts native Go-Redis driver faults into domain apperr classifications.
func NewError(err error, contextMessage string) error {
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
		return apperr.NewNotFound(err, fmt.Sprintf("%s was not found in cache.", contextMessage))
	}

	// Intercept Redis Timeouts / Context Cancellations
	if errors.Is(err, context.DeadlineExceeded) {
		return apperr.NewRequestTimeout(err, "The cache operation timed out.")
	}

	// Default fallback for connection drops or broker failures
	return apperr.NewInternal(err, "A temporary caching error occurred.")
}
