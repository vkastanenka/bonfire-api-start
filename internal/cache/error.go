package cache

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
	ScopeEvents    Scope = "events"
	ScopeCooldown  Scope = "cooldown"
	ScopePresence  Scope = "presence"
	ScopeRateLimit Scope = "rate_limit"
	ScopeSession   Scope = "session"
	ScopeStore     Scope = "store"
	ScopeTicket    Scope = "ticket"
)

func (s Scope) String() string {
	return string(s)
}

var ErrNotFound = errors.New("cache: key not found")

func IsNotFoundError(err error) bool {
	return errors.Is(err, redis.Nil) || errors.Is(err, ErrNotFound)
}

func NewError(err error, scope Scope) error {
	if err == nil {
		return nil
	}

	if appErr := errs.As(err); appErr != nil {
		return err
	}

	if IsNotFoundError(err) {
		return attachContext(
			errs.NotFound(fmt.Sprintf("The requested %s was not found in cache.", scope.String())),
			scope,
		).Wrap(err)
	}

	return handleCacheError(err, scope)
}

func attachContext(e *errs.Error, scope Scope) *errs.Error {
	return e.Meta("scope", scope.String()).Resource("cache", scope.String(), "", "")
}

func handleCacheError(err error, scope Scope) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return attachContext(
			errs.DeadlineExceeded(fmt.Sprintf("Cache operation for %s timed out.", scope.String())),
			scope,
		).Wrap(err)
	}

	if errors.Is(err, context.Canceled) {
		return attachContext(
			errs.Cancelled(fmt.Sprintf("Cache operation for %s was canceled by the client.", scope.String())),
			scope,
		).Wrap(err)
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return attachContext(
			errs.Unavailable(fmt.Sprintf("Cache service for %s is temporarily unavailable.", scope.String())),
			scope,
		).Wrap(err)
	}

	return attachContext(
		errs.Internal(fmt.Sprintf("An internal caching error occurred while processing %s.", scope.String())),
		scope,
	).Wrap(err)
}
