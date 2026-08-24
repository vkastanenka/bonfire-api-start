package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"bonfire-api/internal/fields"
)

// ErrFatal can be wrapped or returned by Handlers to signal a non-retryable failure.
// The worker will immediately move the event to the dead letter state.
var ErrFatal = errors.New("fatal non-retryable outbox error")

// Handler defines the function signature for processing raw outbox payloads.
type Handler func(ctx context.Context, payload json.RawMessage) error

// Repository abstracts the Outbox persistence operations required by the Worker.
type Repository interface {
	ClaimPending(ctx context.Context, workerID fields.ID, leaseExpiresAt fields.Timestamp, now fields.Timestamp, limitVal int) ([]*Event, error)
	Create(ctx context.Context, e *Event) error
	CreateBatch(ctx context.Context, events []*Event) error
	DeleteProcessedBatch(ctx context.Context, before fields.Timestamp, limitVal int) (int64, error)
	MarkDeadLetter(ctx context.Context, e *Event, workerID fields.ID) error
	MarkFailure(ctx context.Context, e *Event, workerID fields.ID) error
	MarkProcessed(ctx context.Context, e *Event, workerID fields.ID) error
	ReleaseLease(ctx context.Context, e *Event, workerID fields.ID) error
	RenewLease(ctx context.Context, e *Event, workerID fields.ID) error
}

// Worker handles polling, concurrent execution, and state management of outbox events.
type Worker struct {
	id            fields.ID
	repo          Repository
	pollInterval  time.Duration
	leaseDuration int
	batchSize     int
	maxWorkers    int
	handlers      map[string]Handler
	handlersMu    sync.RWMutex
	wg            sync.WaitGroup
	cancel        context.CancelFunc
}

func NewWorker(
	repo Repository,
	pollInterval time.Duration,
	leaseDuration int,
	batchSize int,
	maxWorkers int,
) (*Worker, error) {
	if maxWorkers <= 0 {
		maxWorkers = 10
	}
	if leaseDuration <= 0 {
		leaseDuration = 30
	}

	id, err := fields.NewID()
	if err != nil {
		return nil, err
	}

	return &Worker{
		id:            id,
		repo:          repo,
		pollInterval:  pollInterval,
		leaseDuration: leaseDuration,
		batchSize:     batchSize,
		maxWorkers:    maxWorkers,
		handlers:      make(map[string]Handler),
	}, nil
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
		slog.Info("outbox background processor gracefully stopped", "worker_id", w.id)
	}
}

func (w *Worker) processBatch(ctx context.Context) {
	now := fields.Now()
	leaseExpiresAt := fields.NewTimestamp(now.Time().Add(time.Duration(w.leaseDuration) * time.Second))

	events, err := w.repo.ClaimPending(ctx, w.id, leaseExpiresAt, now, w.batchSize)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.ErrorContext(ctx, "failed to acquire outbox events", "error", err)
		}
		return
	}

	if len(events) == 0 {
		return
	}

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
	handler, exists := w.handlers[event.EventType().String()]
	w.handlersMu.RUnlock()

	if !exists {
		slog.WarnContext(ctx, "unhandled event type encountered",
			"event_type", event.EventType(),
			"event_id", event.ID().UUID(),
		)
		w.handleFailure(ctx, event, fmt.Errorf("no handler registered for event type: %s", event.EventType()), true)
		return
	}

	// 1. Calculate execution timeout (80% of lease duration)
	handlerTimeout := time.Duration(float64(w.leaseDuration)*0.8) * time.Second
	if handlerTimeout <= 0 {
		handlerTimeout = 5 * time.Second
	}

	handlerCtx, cancelHandler := context.WithTimeout(ctx, handlerTimeout)
	defer cancelHandler()

	// 2. Inject trace metadata if present on event entity
	if traceID := event.TraceID(); traceID.String() != "" {
		handlerCtx = context.WithValue(handlerCtx, "trace_id", traceID)
	}

	// 3. Heartbeat goroutine: renews lease for long-running handlers
	heartbeatDone := make(chan struct{})
	defer close(heartbeatDone)
	go w.startHeartbeat(ctx, event, heartbeatDone)

	executionErr := handler(handlerCtx, event.Payload().Raw())

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

	// 4. Finalize state using worker ID for verification
	finalizeCtx, cancelFinalize := detachContext(ctx, 3*time.Second)
	defer cancelFinalize()

	if err := w.repo.MarkProcessed(finalizeCtx, event, w.id); err != nil {
		slog.ErrorContext(finalizeCtx, "failed to mark outbox event as processed",
			"event_id", event.ID().UUID(),
			"worker_id", w.id,
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

	// Evaluate domain failure threshold
	if isFatal || (event.Attempts()+1) >= event.MaxAttempts() {
		slog.ErrorContext(logCtx, "outbox event execution exhausted or fatal error; moving to dead letter",
			"event_id", event.ID().UUID(),
			"event_type", event.EventType(),
			"attempts", event.Attempts()+1,
			"max_attempts", event.MaxAttempts(),
			"error", err,
		)

		if dbErr := w.repo.MarkDeadLetter(finalizeCtx, event, w.id); dbErr != nil {
			slog.ErrorContext(finalizeCtx, "failed to dead letter outbox event",
				"event_id", event.ID().UUID(),
				"worker_id", w.id,
				"error", dbErr,
			)
		}
	} else {
		slog.WarnContext(logCtx, "outbox event execution failed; scheduling retry",
			"event_id", event.ID().UUID(),
			"attempt", event.Attempts()+1,
			"error", err,
		)

		if dbErr := w.repo.MarkFailure(finalizeCtx, event, w.id); dbErr != nil {
			slog.ErrorContext(finalizeCtx, "failed to record outbox failure state",
				"event_id", event.ID().UUID(),
				"worker_id", w.id,
				"error", dbErr,
			)
		}
	}
}

// startHeartbeat periodically extends the database lease while processing tasks.
func (w *Worker) startHeartbeat(parentCtx context.Context, event *Event, done <-chan struct{}) {
	interval := time.Duration(w.leaseDuration/2) * time.Second
	if interval < 2*time.Second {
		interval = 2 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-parentCtx.Done():
			return
		case <-ticker.C:
			renewCtx, cancel := detachContext(parentCtx, 2*time.Second)
			if err := w.repo.RenewLease(renewCtx, event, w.id); err != nil {
				slog.WarnContext(renewCtx, "failed to renew outbox event lease",
					"event_id", event.ID().UUID(),
					"worker_id", w.id,
					"error", err,
				)
			}
			cancel()
		}
	}
}

// detachedCtx wraps a parent context to ignore cancellation while preserving value propagation.
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
