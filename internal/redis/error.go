package redis

import (
	"context"
	"errors"
	"fmt"
	"net"

	"bonfire-api/internal/errs"

	"github.com/redis/go-redis/v9"
)

type Scope string

const (
	ScopeAuth      Scope = "auth"
	ScopeChannels  Scope = "channel"
	ScopeEvents    Scope = "events"
	ScopeMessages  Scope = "message"
	ScopePresences Scope = "presence"
	ScopeSessions  Scope = "session"
	ScopeStore     Scope = "store"
	ScopeTickets   Scope = "ticket"
	ScopeUsers     Scope = "user"
)

func (e Scope) String() string { return string(e) }

// ErrCacheMiss represents a standard cache miss and wraps redis.Nil for direct comparison.
var ErrCacheMiss = fmt.Errorf("cache: key not found: %w", redis.Nil)

// IsCacheMiss checks if the underlying error is a redis.Nil or our package sentinel ErrCacheMiss.
func IsCacheMiss(err error) bool {
	return errors.Is(err, redis.Nil) || errors.Is(err, ErrCacheMiss)
}

// NewError transforms raw application/Redis errors into structured domain errors with scope metadata.
func NewError(err error, scope Scope) error {
	if err == nil {
		return nil
	}

	if appErr := errs.As(err); appErr != nil {
		return err
	}

	return handleCacheError(err, scope)
}

func handleCacheError(err error, scope Scope) error {
	var (
		builder func(string) *errs.Error
		msg     string
	)

	switch {
	case IsCacheMiss(err):
		builder = errs.NotFound
		msg = fmt.Sprintf("Cache key for %s not found.", scope)
	case errors.Is(err, context.DeadlineExceeded):
		builder = errs.DeadlineExceeded
		msg = fmt.Sprintf("Cache operation for %s timed out.", scope)
	case errors.Is(err, context.Canceled):
		builder = errs.Cancelled
		msg = fmt.Sprintf("Cache operation for %s was canceled by the client.", scope)
	case isNetworkError(err):
		builder = errs.Unavailable
		msg = fmt.Sprintf("Cache service for %s is temporarily unavailable.", scope)
	default:
		builder = errs.Internal
		msg = fmt.Sprintf("An internal caching error occurred while processing %s.", scope)
	}

	return attachContext(builder(msg), scope).Wrap(err)
}

func isNetworkError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}

func attachContext(e *errs.Error, scope Scope) *errs.Error {
	return e.Meta("scope", scope.String()).Resource("cache", scope.String(), "", "")
}
