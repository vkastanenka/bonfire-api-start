package outbox

import (
	"bonfire-api/internal/fields"
	"encoding/json"
	"time"
)

type Event struct {
	id             fields.ID
	eventType      EventType
	payload        json.RawMessage
	processedAt    fields.Timestamp
	attempts       int32
	maxAttempts    int32
	nextAttemptAt  fields.Timestamp
	lockedBy       fields.ID
	leaseExpiresAt fields.Timestamp
	lastError      LastError
	createdAt      fields.Timestamp
	updatedAt      fields.Timestamp
}

// ============================================================================
// Getters
// ============================================================================

func (e *Event) ID() fields.ID                    { return e.id }
func (e *Event) EventType() EventType             { return e.eventType }
func (e *Event) Payload() json.RawMessage         { return e.payload }
func (e *Event) ProcessedAt() fields.Timestamp    { return e.processedAt }
func (e *Event) Attempts() int32                  { return e.attempts }
func (e *Event) MaxAttempts() int32               { return e.maxAttempts }
func (e *Event) NextAttemptAt() fields.Timestamp  { return e.nextAttemptAt }
func (e *Event) LockedBy() fields.ID              { return e.lockedBy }
func (e *Event) LeaseExpiresAt() fields.Timestamp { return e.leaseExpiresAt }
func (e *Event) LastError() LastError             { return e.lastError }
func (e *Event) CreatedAt() fields.Timestamp      { return e.createdAt }
func (e *Event) UpdatedAt() fields.Timestamp      { return e.updatedAt }

// ============================================================================
// Meta
// ============================================================================

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

// ============================================================================
// Mappers
// ============================================================================

func NewEvent(
	id fields.ID,
	eventType EventType,
	payload json.RawMessage,
	processedAt fields.Timestamp,
	attempts int32,
	maxAttempts int32,
	nextAttemptAt fields.Timestamp,
	lockedBy fields.ID,
	leaseExpiresAt fields.Timestamp,
	lastError LastError,
	createdAt fields.Timestamp,
	updatedAt fields.Timestamp,
) *Event {
	return &Event{
		id:             id,
		eventType:      eventType,
		payload:        payload,
		processedAt:    processedAt,
		attempts:       attempts,
		maxAttempts:    maxAttempts,
		nextAttemptAt:  nextAttemptAt,
		lockedBy:       lockedBy,
		leaseExpiresAt: leaseExpiresAt,
		lastError:      lastError,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}
}

// ============================================================================
// Mutations
// ============================================================================

// Claim updates worker lock ownership and sets the lease duration.
func (e *Event) Claim(workerID fields.ID, leaseExpiresAt fields.Timestamp, at fields.Timestamp) {
	e.lockedBy = workerID
	e.leaseExpiresAt = leaseExpiresAt
	e.updatedAt = at
}

// MarkProcessed transitions the event to a completed state and clears locks.
func (e *Event) MarkProcessed(at fields.Timestamp) {
	e.processedAt = at
	e.lockedBy = fields.ID{}
	e.leaseExpiresAt = fields.Timestamp{}
	e.updatedAt = at
}

// MarkFailure increments attempts, calculates exponential backoff, and releases worker locks.
func (e *Event) MarkFailure(reason string, at fields.Timestamp) {
	e.attempts++
	e.lockedBy = fields.ID{}
	e.leaseExpiresAt = fields.Timestamp{}

	if reason != "" {
		if parsed, err := ParseLastError("last_error", reason); err == nil {
			e.lastError = parsed
		}
	}

	// Exponential backoff: 2^attempts seconds (e.g., 2s, 4s, 8s, 16s...)
	backoffSec := time.Duration(1<<e.attempts) * time.Second
	e.nextAttemptAt = fields.NewTimestampFromTime(at.Time().Add(backoffSec))
	e.updatedAt = at
}

// MarkDeadLetter maxes out attempts and parks the event without scheduling future attempts.
func (e *Event) MarkDeadLetter(reason string, at fields.Timestamp) {
	e.attempts = e.maxAttempts
	e.lockedBy = fields.ID{}
	e.leaseExpiresAt = fields.Timestamp{}

	if reason != "" {
		if parsed, err := ParseLastError("last_error", reason); err == nil {
			e.lastError = parsed
		}
	}

	e.updatedAt = at
}

// RenewLease extends the worker's lock reservation time.
func (e *Event) RenewLease(newLeaseExpiresAt fields.Timestamp, at fields.Timestamp) {
	e.leaseExpiresAt = newLeaseExpiresAt
	e.updatedAt = at
}

// ReleaseLease explicitly clears worker ownership without altering retry counts or errors.
func (e *Event) ReleaseLease(at fields.Timestamp) {
	e.lockedBy = fields.ID{}
	e.leaseExpiresAt = fields.Timestamp{}
	e.updatedAt = at
}
