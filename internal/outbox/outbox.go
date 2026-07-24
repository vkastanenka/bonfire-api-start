package outbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Status uint8

const (
	StatusUnknown    Status = 0
	StatusPending    Status = 1
	StatusProcessing Status = 2
	StatusProcessed  Status = 3
	StatusDeadLetter Status = 4
)

var statusNames = [...]string{
	StatusUnknown:    "UNKNOWN",
	StatusPending:    "PENDING",
	StatusProcessing: "PROCESSING",
	StatusProcessed:  "PROCESSED",
	StatusDeadLetter: "DEAD_LETTER",
}

func (s Status) Valid() bool {
	return int(s) > int(StatusUnknown) && int(s) < len(statusNames)
}

func (s Status) String() string {
	if int(s) >= 0 && int(s) < len(statusNames) {
		return statusNames[s]
	}
	return fmt.Sprintf("STATUS_%d", s)
}

var (
	ErrEmptyEventType = errors.New("event type cannot be empty")
	ErrEmptyPayload   = errors.New("event payload cannot be empty")
	ErrAlreadyLocked  = errors.New("outbox event is already locked by another worker")
	ErrLeaseExpired   = errors.New("outbox event lease has expired or worker mismatch")
)

type EventID struct {
	value uuid.UUID
}

func NewEventID(id uuid.UUID) (EventID, error) {
	if id == uuid.Nil {
		return EventID{}, errors.New("event ID cannot be empty")
	}
	return EventID{value: id}, nil
}

func (id EventID) UUID() uuid.UUID { return id.value }

type Event struct {
	id             EventID
	eventType      string
	payload        json.RawMessage
	processedAt    *time.Time
	attempts       int32
	maxAttempts    int32
	nextAttemptAt  time.Time
	lockedBy       *uuid.UUID
	leaseExpiresAt *time.Time
	lastError      *string
	createdAt      time.Time
	updatedAt      time.Time
}

// Getters (Encapsulation)
func (e *Event) ID() EventID              { return e.id }
func (e *Event) EventType() string        { return e.eventType }
func (e *Event) Payload() json.RawMessage { return e.payload }
func (e *Event) ProcessedAt() *time.Time  { return e.processedAt }
func (e *Event) Attempts() int32          { return e.attempts }
func (e *Event) MaxAttempts() int32       { return e.maxAttempts }
func (e *Event) NextAttemptAt() time.Time { return e.nextAttemptAt }
func (e *Event) LockedBy() *uuid.UUID     { return e.lockedBy }
func (e *Event) LeaseExpiresAt() *time.Time {
	if e.leaseExpiresAt == nil {
		return nil
	}
	t := *e.leaseExpiresAt
	return &t
}
func (e *Event) LastError() *string   { return e.lastError }
func (e *Event) CreatedAt() time.Time { return e.createdAt }
func (e *Event) UpdatedAt() time.Time { return e.updatedAt }

func (e *Event) IsProcessed() bool  { return e.processedAt != nil }
func (e *Event) IsDeadLetter() bool { return e.attempts >= e.maxAttempts }

func (e *Event) Status() Status {
	if e.IsProcessed() {
		return StatusProcessed
	}
	if e.IsDeadLetter() {
		return StatusDeadLetter
	}
	if e.leaseExpiresAt != nil && e.leaseExpiresAt.After(time.Now()) {
		return StatusProcessing
	}
	return StatusPending
}

// New constructs a new outbox event aggregate
func New(eventType string, payload json.RawMessage) (*Event, error) {
	if strings.TrimSpace(eventType) == "" {
		return nil, ErrEmptyEventType
	}
	if len(payload) == 0 {
		return nil, ErrEmptyPayload
	}
	// if maxAttempts <= 0 {
	// 	maxAttempts = 5 // Reasonable domain default
	// }

	now := time.Now().UTC()
	id, err := NewEventID(uuid.Must(uuid.NewV7()))
	if err != nil {
		return nil, err
	}

	return &Event{
		id:            id,
		eventType:     eventType,
		payload:       payload,
		attempts:      0,
		maxAttempts:   5,
		nextAttemptAt: now,
		createdAt:     now,
		updatedAt:     now,
	}, nil
}

// Reconstitute restores an Event from persistence without triggering domain side-effects
func Reconstitute(
	id EventID,
	eventType string,
	payload json.RawMessage,
	processedAt *time.Time,
	attempts, maxAttempts int32,
	nextAttemptAt time.Time,
	lockedBy *uuid.UUID,
	leaseExpiresAt *time.Time,
	lastError *string,
	createdAt, updatedAt time.Time,
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

// State transition domain methods

func (e *Event) MarkProcessed(at time.Time) {
	t := at.UTC()
	e.processedAt = &t
	e.lockedBy = nil
	e.leaseExpiresAt = nil
	e.updatedAt = t
}

func (e *Event) RecordFailure(errReason string, at time.Time) {
	now := at.UTC()
	e.attempts++
	e.lockedBy = nil
	e.leaseExpiresAt = nil
	if errReason != "" {
		e.lastError = &errReason
	}

	// Exponential backoff strategy encapsulated in the domain
	backoffSeconds := math.Pow(2, float64(e.attempts))
	e.nextAttemptAt = now.Add(time.Duration(backoffSeconds) * time.Second)
	e.updatedAt = now
}

func (e *Event) MarkDeadLetter(reason string, at time.Time) {
	now := at.UTC()
	e.attempts = e.maxAttempts
	e.lockedBy = nil
	e.leaseExpiresAt = nil
	if reason != "" {
		e.lastError = &reason
	}
	e.updatedAt = now
}

func (e *Event) ResetAttempts(at time.Time) {
	now := at.UTC()
	e.attempts = 0
	e.lockedBy = nil
	e.leaseExpiresAt = nil
	e.lastError = nil
	e.nextAttemptAt = now
	e.updatedAt = now
}
