package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ErrFatal can be wrapped or returned by Handlers to signal a non-retryable failure
// (e.g., corrupted JSON, business rule violation). The worker will immediately DLQ the event.
var ErrFatal = errors.New("fatal non-retryable outbox error")

// Handler defines the function signature for processing raw outbox payloads.
type Handler func(ctx context.Context, payload json.RawMessage) error

// Repository abstracts the Outbox persistence operations required by the Worker.
type Repository interface {
	AcquireBatch(ctx context.Context, workerID uuid.UUID, leaseDurationSec, batchSize int32) ([]*Event, error)
	MarkProcessed(ctx context.Context, id EventID) (*Event, error)
	RecordFailure(ctx context.Context, id EventID, lastError string) (*Event, error)
	MarkDeadLetter(ctx context.Context, id EventID, reason string) (*Event, error)
}

// Worker handles polling, concurrent execution, and state management of outbox events.
type Worker struct {
	id            uuid.UUID
	repo          Repository
	pollInterval  time.Duration
	leaseDuration int32
	batchSize     int32
	maxWorkers    int
	handlers      map[string]Handler
	handlersMu    sync.RWMutex
	wg            sync.WaitGroup
	cancel        context.CancelFunc
}

func NewWorker(
	repo Repository,
	pollInterval time.Duration,
	leaseDuration int32,
	batchSize int32,
	maxWorkers int,
) *Worker {
	if maxWorkers <= 0 {
		maxWorkers = 10
	}

	return &Worker{
		id:            uuid.New(),
		repo:          repo,
		pollInterval:  pollInterval,
		leaseDuration: leaseDuration,
		batchSize:     batchSize,
		maxWorkers:    maxWorkers,
		handlers:      make(map[string]Handler),
	}
}

// RegisterHandler registers a callback function for a specific event type.
func (w *Worker) RegisterHandler(eventType string, handler Handler) {
	w.handlersMu.Lock()
	defer w.handlersMu.Unlock()
	w.handlers[eventType] = handler
}

// Start launches the background worker polling loop.
func (w *Worker) Start(ctx context.Context) {
	workerCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(workerCtx, "recovered from panic in outbox worker loop", "panic", r)
			}
		}()

		slog.InfoContext(workerCtx, "initializing outbox background processor",
			"worker_id", w.id,
			"batch_size", w.batchSize,
			"poll_interval", w.pollInterval,
			"max_workers", w.maxWorkers,
		)

		// Run immediately on startup
		w.processBatch(workerCtx)

		ticker := time.NewTicker(w.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				w.processBatch(workerCtx)
			case <-workerCtx.Done():
				slog.InfoContext(workerCtx, "stopping outbox worker loop")
				return
			}
		}
	}()
}

// Stop gracefully waits for in-flight tasks to complete before shutting down.
func (w *Worker) Stop() {
	if w.cancel != nil {
		w.cancel()
		w.wg.Wait()
		slog.Info("outbox background processor gracefully stopped")
	}
}

func (w *Worker) processBatch(ctx context.Context) {
	events, err := w.repo.AcquireBatch(ctx, w.id, w.leaseDuration, w.batchSize)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.ErrorContext(ctx, "failed to acquire outbox events", "error", err)
		}
		return
	}

	if len(events) == 0 {
		return
	}

	// Worker pool mechanism: bound concurrent execution so we don't saturate resources
	sem := make(chan struct{}, w.maxWorkers)
	var batchWg sync.WaitGroup

	for _, evt := range events {
		if ctx.Err() != nil {
			break
		}

		batchWg.Add(1)
		sem <- struct{}{} // Acquire semaphore slot

		go func(e *Event) {
			defer batchWg.Done()
			defer func() { <-sem }() // Release semaphore slot

			w.executeEvent(ctx, e)
		}(evt)
	}

	batchWg.Wait()
}

func (w *Worker) executeEvent(ctx context.Context, event *Event) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "recovered from panic during outbox event execution",
				"event_id", event.ID().UUID(),
				"event_type", event.EventType(),
				"panic", r,
			)
			w.handleFailure(ctx, event, fmt.Errorf("panic during execution: %v", r), true)
		}
	}()

	w.handlersMu.RLock()
	handler, exists := w.handlers[event.EventType()]
	w.handlersMu.RUnlock()

	if !exists {
		slog.WarnContext(ctx, "unhandled event type encountered",
			"event_type", event.EventType(),
			"event_id", event.ID().UUID(),
		)
		w.handleFailure(ctx, event, fmt.Errorf("no handler registered for event type: %s", event.EventType()), true)
		return
	}

	// Calculate execution timeout (80% of lease duration) so handlers cannot hang indefinitely
	handlerTimeout := time.Duration(float64(w.leaseDuration)*0.8) * time.Second
	if handlerTimeout <= 0 {
		handlerTimeout = 5 * time.Second
	}

	handlerCtx, cancelHandler := context.WithTimeout(ctx, handlerTimeout)
	defer cancelHandler()

	executionErr := handler(handlerCtx, event.Payload())

	if executionErr != nil {
		if errors.Is(executionErr, context.Canceled) && ctx.Err() != nil {
			slog.InfoContext(ctx, "execution context canceled during shutdown; leaving lease to expire for recovery",
				"event_id", event.ID().UUID(),
			)
			return
		}

		isFatal := errors.Is(executionErr, ErrFatal)
		w.handleFailure(ctx, event, executionErr, isFatal)
		return
	}

	// Detached context ensures finalize DB calls succeed during application shutdown while preserving trace values
	finalizeCtx, cancelFinalize := detachContext(ctx, 3*time.Second)
	defer cancelFinalize()

	if _, err := w.repo.MarkProcessed(finalizeCtx, event.ID()); err != nil {
		slog.ErrorContext(finalizeCtx, "failed to mark outbox event as processed",
			"event_id", event.ID().UUID(),
			"error", err,
		)
		return
	}

	slog.DebugContext(finalizeCtx, "successfully processed outbox event",
		"event_id", event.ID().UUID(),
		"event_type", event.EventType(),
	)
}

func (w *Worker) handleFailure(ctx context.Context, event *Event, err error, isFatal bool) {
	finalizeCtx, cancel := detachContext(ctx, 3*time.Second)
	defer cancel()

	logCtx := ctx
	if logCtx.Err() != nil {
		logCtx = finalizeCtx
	}

	// Check domain rules: if explicitly fatal OR next increment reaches max attempts
	if isFatal || (event.Attempts()+1) >= event.MaxAttempts() {
		slog.ErrorContext(logCtx, "outbox event execution exhausted or fatal error; moving to dead letter",
			"event_id", event.ID().UUID(),
			"event_type", event.EventType(),
			"attempts", event.Attempts()+1,
			"max_attempts", event.MaxAttempts(),
			"error", err,
		)

		if _, dbErr := w.repo.MarkDeadLetter(finalizeCtx, event.ID(), err.Error()); dbErr != nil {
			slog.ErrorContext(finalizeCtx, "failed to dead letter outbox event",
				"event_id", event.ID().UUID(),
				"error", dbErr,
			)
		}
	} else {
		slog.WarnContext(logCtx, "outbox event execution failed; scheduling retry",
			"event_id", event.ID().UUID(),
			"attempt", event.Attempts()+1,
			"error", err,
		)

		if _, dbErr := w.repo.RecordFailure(finalizeCtx, event.ID(), err.Error()); dbErr != nil {
			slog.ErrorContext(finalizeCtx, "failed to record outbox failure state",
				"event_id", event.ID().UUID(),
				"error", dbErr,
			)
		}
	}
}

// detachedCtx wraps a parent context to ignore cancellation while preserving key/value context metadata.
type detachedCtx struct {
	parent context.Context
}

func (d detachedCtx) Deadline() (time.Time, bool) { return time.Time{}, false }
func (d detachedCtx) Done() <-chan struct{}       { return nil }
func (d detachedCtx) Err() error                  { return nil }
func (d detachedCtx) Value(key any) any           { return d.parent.Value(key) }

func detachContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(detachedCtx{parent: ctx}, timeout)
}
