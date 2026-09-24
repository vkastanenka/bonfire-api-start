package redis

import (
	"context"
	"errors"
	"net"

	"bonfire-api/internal/pkg/errs"

	"github.com/redis/go-redis/v9"
)

type Scope string

const (
	ScopeAuth        Scope = "auth"
	ScopeChannel     Scope = "channel"
	ScopeGateway     Scope = "gateway"
	ScopeMember      Scope = "member"
	ScopeOutboxEvent Scope = "outbox_event"
	ScopeMessage     Scope = "message"
	ScopeReaction    Scope = "reaction"
	ScopePresence    Scope = "presence"
	ScopeRelation    Scope = "relation"
	ScopeSession     Scope = "session"
	ScopeStore       Scope = "store"
	ScopeTicket      Scope = "ticket"
	ScopeToken       Scope = "token"
	ScopeUser        Scope = "user"
)

func (e Scope) String() string { return string(e) }

var (
	// ErrCacheMiss indicates the requested key does not exist in Redis.
	ErrCacheMiss = errors.New("cache: key not found")

	// ErrCorruptedData indicates raw bytes were found but failed deserialization.
	ErrCorruptedData = errors.New("cache: data corrupted or schema mismatch")
)

// IsCacheMiss checks if the underlying error is a redis.Nil or package sentinel ErrCacheMiss.
func IsCacheMiss(err error) bool {
	return errors.Is(err, redis.Nil) || errors.Is(err, ErrCacheMiss)
}

// NewError transforms raw application/Redis errors into structured domain errors with scope metadata.
func NewError(err error, scope Scope) error {
	if err == nil || IsCacheMiss(err) {
		return nil
	}

	if appErr := errs.As(err); appErr != nil {
		return err
	}

	return handleCacheError(err, scope)
}

func handleCacheError(err error, scope Scope) error {
	switch {
	case errors.Is(err, ErrCorruptedData):
		return errs.Internal("Cached data is corrupted.").
			Reason("CACHE_CORRUPTED").
			Meta("scope", scope.String()).
			Wrap(err)

	case errors.Is(err, context.DeadlineExceeded):
		return errs.DeadlineExceeded("Cache operation timed out.").
			Reason("CACHE_TIMEOUT").
			Meta("scope", scope.String()).
			Wrap(err)

	case errors.Is(err, context.Canceled):
		return errs.Aborted("Cache operation was canceled.").
			Reason("CACHE_CANCELED").
			Meta("scope", scope.String()).
			Wrap(err)

	case isNetworkError(err):
		return errs.Unavailable("Cache service is temporarily unavailable.").
			Reason("CACHE_UNAVAILABLE").
			Meta("scope", scope.String()).
			Wrap(err)

	default:
		return errs.Internal("An internal caching error occurred.").
			Reason("CACHE_INTERNAL_ERROR").
			Meta("scope", scope.String()).
			Wrap(err)
	}
}

func isNetworkError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}
