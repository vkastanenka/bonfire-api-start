package outbox

import (
	"context"
	"encoding/json"
	"time"

	"bonfire-api/internal/httpio"

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
		LockedBy:       lockedBy,
		LeaseExpiresAt: leaseExpiresAt,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
}

func New(
	ctx context.Context,
	eventType string,
	payload json.RawMessage,
	now time.Time,
) (*Event, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	var tracePtr *string
	if tid := httpio.CtxGetTraceID(ctx); tid != "" {
		tracePtr = &tid
	}

	return ReconstituteEvent(
		id,
		eventType,
		payload,
		tracePtr,
		nil,
		0,
		maxAttemptsDefault,
		now,
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

// MarkFailure increments attempts, calculates exponential backoff, and releases worker locks.
func (e *Event) MarkFailure(at time.Time) {
	e.Attempts++
	e.LockedBy = nil
	e.LeaseExpiresAt = nil

	// Exponential backoff: 2^attempts seconds (e.g., 2s, 4s, 8s, 16s...)
	backoffSec := time.Duration(1<<e.Attempts) * time.Second
	e.NextAttemptAt = at.Add(backoffSec)
	e.touch(at)
}

// MarkDeadLetter maxes out attempts and parks the event without scheduling future attempts.
func (e *Event) MarkDeadLetter(at time.Time) {
	e.Attempts = e.MaxAttempts
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
