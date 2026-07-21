package cache

import (
	"bonfire-api/internal/apperr"
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type Scope string

const (
	ScopeAuth      Scope = "auth"
	ScopeEvents    Scope = "events"
	ScopePresence  Scope = "presence"
	ScopeRateLimit Scope = "rate_limit"
	ScopeSession   Scope = "session"
	ScopeStore     Scope = "store"
)

func (r Scope) String() string {
	return string(r)
}

var ErrNotFound = errors.New("cache: key not found")

func IsNotFoundError(err error) bool {
	return errors.Is(err, redis.Nil) || errors.Is(err, ErrNotFound)
}

func NewError(err error, resource Scope) error {
	if err == nil {
		return nil
	}

	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		return err
	}

	if IsNotFoundError(err) {
		return apperr.NewNotFound(err, apperr.WithMsg(fmt.Sprintf("%s was not found in cache.", resource)))
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return apperr.NewDeadlineExceeded(err, apperr.WithMsg("The cache operation timed out.")) // request timeout
	}
	if errors.Is(err, context.Canceled) {
		return apperr.NewInvalidArgument(err, apperr.WithMsg("The cache transaction was aborted by the client."))
	}

	return apperr.NewInternal(err, apperr.WithMsg("A temporary caching error occurred."))
}
