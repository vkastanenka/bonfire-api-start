package outbox

import (
	"context"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/httpio"
)

const maxAttemptsDefault = 5

type Event struct {
	id             fields.ID
	eventType      Type
	payload        Payload
	traceID        fields.TraceID
	processedAt    fields.Timestamp
	attempts       int
	maxAttempts    int
	nextAttemptAt  fields.Timestamp
	lockedBy       fields.ID
	leaseExpiresAt fields.Timestamp
	createdAt      fields.Timestamp
	updatedAt      fields.Timestamp
}

func ReconstituteEvent(
	id fields.ID,
	eventType Type,
	payload Payload,
	traceID fields.TraceID,
	processedAt fields.Timestamp,
	attempts int,
	maxAttempts int,
	nextAttemptAt fields.Timestamp,
	lockedBy fields.ID,
	leaseExpiresAt fields.Timestamp,
	createdAt fields.Timestamp,
	updatedAt fields.Timestamp,
) *Event {
	return &Event{
		id:             id,
		eventType:      eventType,
		payload:        payload,
		traceID:        traceID,
		processedAt:    processedAt,
		attempts:       attempts,
		maxAttempts:    maxAttempts,
		nextAttemptAt:  nextAttemptAt,
		lockedBy:       lockedBy,
		leaseExpiresAt: leaseExpiresAt,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}
}

func New(
	ctx context.Context,
	eventType Type,
	payload Payload,
	now fields.Timestamp,
) (*Event, error) {
	id, err := fields.NewID()
	if err != nil {
		return nil, err
	}

	traceID := httpio.CtxGetTraceID(ctx)

	return ReconstituteEvent(
		id,
		eventType,
		payload,
		traceID,
		fields.Timestamp{},
		0,
		maxAttemptsDefault,
		now,
		fields.ID{},
		fields.Timestamp{},
		now,
		now,
	), nil
}

func (e *Event) ID() fields.ID                    { return e.id }
func (e *Event) EventType() Type                  { return e.eventType }
func (e *Event) Payload() Payload                 { return e.payload }
func (e *Event) TraceID() fields.TraceID          { return e.traceID }
func (e *Event) ProcessedAt() fields.Timestamp    { return e.processedAt }
func (e *Event) Attempts() int                    { return e.attempts }
func (e *Event) MaxAttempts() int                 { return e.maxAttempts }
func (e *Event) NextAttemptAt() fields.Timestamp  { return e.nextAttemptAt }
func (e *Event) LockedBy() fields.ID              { return e.lockedBy }
func (e *Event) LeaseExpiresAt() fields.Timestamp { return e.leaseExpiresAt }
func (e *Event) CreatedAt() fields.Timestamp      { return e.createdAt }
func (e *Event) UpdatedAt() fields.Timestamp      { return e.updatedAt }

func (e *Event) IsProcessed() bool  { return e.processedAt.IsValid() }
func (e *Event) IsDeadLetter() bool { return e.attempts >= e.maxAttempts }
func (e *Event) IsLocked() bool     { return e.lockedBy.IsValid() }

func (e *Event) CanProcess(now fields.Timestamp) bool {
	if e.IsProcessed() || e.IsDeadLetter() {
		return false
	}
	if e.nextAttemptAt.HasPassed(now.Time()) {
		return true
	}
	return e.nextAttemptAt.Equals(now)
}

// Claim updates worker lock ownership and sets the lease duration.
func (e *Event) Claim(workerID fields.ID, leaseExpiresAt fields.Timestamp, at fields.Timestamp) {
	e.lockedBy = workerID
	e.leaseExpiresAt = leaseExpiresAt
	e.touch(at)
}

// MarkProcessed transitions the event to a completed state and clears locks.
func (e *Event) MarkProcessed(at fields.Timestamp) {
	e.processedAt = at
	e.lockedBy = fields.ID{}
	e.leaseExpiresAt = fields.Timestamp{}
	e.touch(at)
}

// MarkFailure increments attempts, calculates exponential backoff, and releases worker locks.
func (e *Event) MarkFailure(reason string, at fields.Timestamp) {
	e.attempts++
	e.lockedBy = fields.ID{}
	e.leaseExpiresAt = fields.Timestamp{}

	// Exponential backoff: 2^attempts seconds (e.g., 2s, 4s, 8s, 16s...)
	backoffSec := time.Duration(1<<e.attempts) * time.Second
	e.nextAttemptAt = fields.NewTimestamp(at.Time().Add(backoffSec))
	e.touch(at)
}

// MarkDeadLetter maxes out attempts and parks the event without scheduling future attempts.
func (e *Event) MarkDeadLetter(reason string, at fields.Timestamp) {
	e.attempts = e.maxAttempts
	e.lockedBy = fields.ID{}
	e.leaseExpiresAt = fields.Timestamp{}

	e.touch(at)
}

// RenewLease extends the worker's lock reservation time.
func (e *Event) RenewLease(newLeaseExpiresAt fields.Timestamp, at fields.Timestamp) {
	e.leaseExpiresAt = newLeaseExpiresAt
	e.touch(at)
}

// ReleaseLease explicitly clears worker ownership without altering retry counts or errors.
func (e *Event) ReleaseLease(at fields.Timestamp) {
	e.lockedBy = fields.ID{}
	e.leaseExpiresAt = fields.Timestamp{}
	e.touch(at)
}

func (e *Event) touch(at fields.Timestamp) {
	e.updatedAt = at
}
