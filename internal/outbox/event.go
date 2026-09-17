package outbox

import (
	"context"
	"encoding/json"
	"time"

	"bonfire-api/internal/httpio"
	"bonfire-api/internal/pkg/errs"

	"github.com/google/uuid"
)

const maxAttemptsDefault = 5

type Event struct {
	ID             uuid.UUID
	Type           string
	Payload        json.RawMessage
	TraceID        *string
	ProcessedAt    *time.Time
	Attempts       int
	MaxAttempts    int
	NextAttemptAt  time.Time
	LastError      *string
	LockedBy       *uuid.UUID
	LeaseExpiresAt *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func ReconstituteEvent(
	id uuid.UUID,
	eventType string,
	payload json.RawMessage,
	traceID *string,
	processedAt *time.Time,
	attempts int,
	maxAttempts int,
	nextAttemptAt time.Time,
	lastError *string,
	lockedBy *uuid.UUID,
	leaseExpiresAt *time.Time,
	createdAt time.Time,
	updatedAt time.Time,
) *Event {
	return &Event{
		ID:             id,
		Type:           eventType,
		Payload:        payload,
		TraceID:        traceID,
		ProcessedAt:    processedAt,
		Attempts:       attempts,
		MaxAttempts:    maxAttempts,
		NextAttemptAt:  nextAttemptAt,
		LastError:      lastError,
		LockedBy:       lockedBy,
		LeaseExpiresAt: leaseExpiresAt,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
}

// New constructs a new outbox Event domain entity.
func New(
	ctx context.Context,
	eventType string,
	payload any,
	now time.Time,
) (*Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, errs.Internal("Failed to marshal outbox event payload.").
			Meta("event_type", eventType).
			Wrap(err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, errs.Internal("Failed to generate outbox event ID.").Wrap(err)
	}

	var tracePtr *string
	if tid := httpio.CtxGetTraceID(ctx); tid != "" {
		tracePtr = &tid
	}

	return ReconstituteEvent(
		id,
		eventType,
		data,
		tracePtr,
		nil,
		0,
		maxAttemptsDefault,
		now,
		nil,
		nil,
		nil,
		now,
		now,
	), nil
}

func (e *Event) IsProcessed() bool  { return e.ProcessedAt != nil }
func (e *Event) IsDeadLetter() bool { return e.Attempts >= e.MaxAttempts }
func (e *Event) IsLocked() bool     { return e.LockedBy != nil }

func (e *Event) CanProcess(now time.Time) bool {
	if e.IsProcessed() || e.IsDeadLetter() {
		return false
	}
	return !e.NextAttemptAt.After(now)
}

// Claim updates worker lock ownership and sets the lease duration.
func (e *Event) Claim(workerID uuid.UUID, leaseExpiresAt time.Time, at time.Time) {
	e.LockedBy = &workerID
	e.LeaseExpiresAt = &leaseExpiresAt
	e.touch(at)
}

// MarkProcessed transitions the event to a completed state and clears locks.
func (e *Event) MarkProcessed(at time.Time) {
	e.ProcessedAt = &at
	e.LockedBy = nil
	e.LeaseExpiresAt = nil
	e.touch(at)
}

// MarkFailure increments attempts, records the error, calculates exponential backoff, and releases worker locks.
func (e *Event) MarkFailure(execErr error, at time.Time) {
	e.Attempts++
	e.LastError = formatLastError(execErr)
	e.LockedBy = nil
	e.LeaseExpiresAt = nil

	// Exponential backoff: 2^attempts seconds (e.g., 2s, 4s, 8s, 16s...)
	backoffSec := time.Duration(1<<e.Attempts) * time.Second
	e.NextAttemptAt = at.Add(backoffSec)
	e.touch(at)
}

// MarkDeadLetter maxes out attempts, records the error, and parks the event without scheduling future attempts.
func (e *Event) MarkDeadLetter(execErr error, at time.Time) {
	e.Attempts = e.MaxAttempts
	e.LastError = formatLastError(execErr)
	e.LockedBy = nil
	e.LeaseExpiresAt = nil

	e.touch(at)
}

// RenewLease extends the worker's lock reservation time.
func (e *Event) RenewLease(newLeaseExpiresAt time.Time, at time.Time) {
	e.LeaseExpiresAt = &newLeaseExpiresAt
	e.touch(at)
}

// ReleaseLease explicitly clears worker ownership without altering retry counts or errors.
func (e *Event) ReleaseLease(at time.Time) {
	e.LockedBy = nil
	e.LeaseExpiresAt = nil
	e.touch(at)
}

func (e *Event) touch(at time.Time) {
	e.UpdatedAt = at
}

func formatLastError(err error) *string {
	if err == nil {
		return nil
	}
	errStr := err.Error()
	if len(errStr) > maxLastErrorLen {
		errStr = errStr[:maxLastErrorLen-1] + "..."
	}
	return &errStr
}
