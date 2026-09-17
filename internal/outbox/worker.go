package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"bonfire-api/internal/httpio"
	"bonfire-api/internal/pkg/errs"

	"github.com/google/uuid"
)

var ErrFatal = errors.New("outbox: fatal event execution error")

type Handler func(ctx context.Context, payload json.RawMessage) error

type Worker struct {
	id            uuid.UUID
	repo          Repository
	pollInterval  time.Duration
	leaseDuration time.Duration
	batchSize     int
	maxWorkers    int
	sem           chan struct{}
	handlers      map[string]Handler
	handlersMu    sync.RWMutex
	wg            sync.WaitGroup
	cancel        context.CancelFunc
}

func NewWorker(
	repo Repository,
	pollInterval time.Duration,
	leaseDuration time.Duration,
	batchSize int,
	maxWorkers int,
) (*Worker, error) {
	if repo == nil {
		return nil, errs.Internal("Outbox worker repository cannot be nil.")
	}

	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	if leaseDuration <= 0 {
		leaseDuration = 30 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 50
	}
	if maxWorkers <= 0 {
		maxWorkers = 10
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, errs.Internal("Failed to generate outbox worker ID.").Wrap(err)
	}

	return &Worker{
		id:            id,
		repo:          repo,
		pollInterval:  pollInterval,
		leaseDuration: leaseDuration,
		batchSize:     batchSize,
		maxWorkers:    maxWorkers,
		sem:           make(chan struct{}, maxWorkers), // Initialized once with maxWorkers capacity
		handlers:      make(map[string]Handler),
	}, nil
}

// RegisterHandler registers a callback function for a specific event type.
func (w *Worker) RegisterHandler(eventType string, handler Handler) {
	if handler == nil {
		panic(fmt.Sprintf("outbox worker: handler for event %q cannot be nil", eventType))
	}

	w.handlersMu.Lock()
	defer w.handlersMu.Unlock()

	if _, exists := w.handlers[eventType]; exists {
		panic(fmt.Sprintf("outbox worker: handler for event %q is already registered", eventType))
	}

	w.handlers[eventType] = handler
}

// Start launches the background worker polling loop.
func (w *Worker) Start(ctx context.Context) {
	workerCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

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
				slog.InfoContext(workerCtx, "stopping outbox worker loop", "worker_id", w.id)
				return
			}
		}
	}()
}

// Stop gracefully waits for in-flight tasks to complete before shutting down.
func (w *Worker) Stop() {
	if w.cancel != nil {
		w.cancel()

		done := make(chan struct{})
		go func() {
			w.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			slog.Info("outbox background processor gracefully stopped", "worker_id", w.id)
		case <-time.After(10 * time.Second):
			slog.Warn("outbox background processor shutdown timed out", "worker_id", w.id)
		}
	}
}

// processBatch claims a batch of pending events and dispatches them concurrently across worker routines.
func (w *Worker) processBatch(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "recovered from panic in outbox batch processing",
				"worker_id", w.id,
				"panic", r,
			)
		}
	}()

	now := time.Now()
	leaseExpiresAt := now.Add(w.leaseDuration)

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

	var batchWg sync.WaitGroup

	for _, evt := range events {
		batchWg.Add(1)
		w.sem <- struct{}{}

		go func(e *Event) {
			defer batchWg.Done()
			defer func() { <-w.sem }()

			execCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
			defer cancel()

			w.executeEvent(execCtx, e)
		}(evt)
	}

	batchWg.Wait()
}

// executeEvent processes a single outbox event with timeout safeguards, trace context propagation, and failure handling.
func (w *Worker) executeEvent(ctx context.Context, event *Event) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "recovered from panic during outbox event execution",
				"event_id", event.ID,
				"event_type", event.Type,
				"panic", r,
			)
			w.handleFailure(ctx, event, fmt.Errorf("panic during execution: %v", r), true)
		}
	}()

	w.handlersMu.RLock()
	handler, exists := w.handlers[event.Type]
	w.handlersMu.RUnlock()

	if !exists {
		slog.WarnContext(ctx, "unhandled event type encountered",
			"event_type", event.Type,
			"event_id", event.ID,
		)
		w.handleFailure(ctx, event, fmt.Errorf("no handler registered for event type: %s", event.Type), true)
		return
	}

	// Calculate handler execution timeout boundary
	handlerTimeout := time.Duration(float64(w.leaseDuration) * 0.8)
	if handlerTimeout <= 0 {
		handlerTimeout = 5 * time.Second
	}

	// Construct a base timeout context for the handler
	baseCtx, cancelTimeout := context.WithTimeout(ctx, handlerTimeout)
	defer cancelTimeout()

	// Wrap with WithCancelCause so startHeartbeat can cancel taskCtx on lease loss
	taskCtx, cancelTask := context.WithCancelCause(baseCtx)
	defer cancelTask(nil)

	if event.TraceID != nil {
		taskCtx = context.WithValue(taskCtx, httpio.CtxKeyTraceID, *event.TraceID)
	}

	// Launch background heartbeat using parent ctx (survives taskCtx cancel) and pass cancelTask cause handle
	heartbeatDone := make(chan struct{})
	go w.startHeartbeat(ctx, event, heartbeatDone, cancelTask)

	executionErr := handler(taskCtx, event.Payload)

	// Signal heartbeat goroutine to stop immediately once handler returns
	close(heartbeatDone)

	if executionErr != nil {
		// Check if cancellation was triggered by lost database lease
		if leaseErr := context.Cause(taskCtx); leaseErr != nil && !errors.Is(leaseErr, taskCtx.Err()) {
			slog.ErrorContext(ctx, "handler execution aborted due to lost outbox lease",
				"event_id", event.ID,
				"error", leaseErr,
			)
			// Return without mutating DB state; another worker has reclaimed or will reclaim the event
			return
		}

		// Graceful worker shutdown: leave lease to expire naturally for another worker
		if errors.Is(executionErr, context.Canceled) && ctx.Err() != nil {
			slog.InfoContext(ctx, "execution context canceled during shutdown; leaving lease to expire for recovery",
				"event_id", event.ID,
			)
			return
		}

		isFatal := errors.Is(executionErr, ErrFatal)
		w.handleFailure(ctx, event, executionErr, isFatal)
		return
	}

	// Update domain entity state
	event.MarkProcessed(time.Now())

	// Finalize status update in DB detached from handler execution cancellation
	finalizeCtx, cancelFinalize := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancelFinalize()

	if err := w.repo.MarkProcessed(finalizeCtx, event, w.id); err != nil {
		slog.ErrorContext(finalizeCtx, "failed to mark outbox event as processed",
			"event_id", event.ID,
			"worker_id", w.id,
			"error", err,
		)
		return
	}

	slog.DebugContext(finalizeCtx, "successfully processed outbox event",
		"event_id", event.ID,
		"event_type", event.Type,
	)
}

// handleFailure transitions an event to either a dead-letter state or schedules a retry with backoff.
func (w *Worker) handleFailure(ctx context.Context, event *Event, executionErr error, isFatal bool) {
	now := time.Now()

	// Detached context ensures persistence finishes during SIGTERM / graceful shutdown
	finalizeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()

	// Capture initial attempt before state transitions for accurate logging
	currentAttempt := event.Attempts + 1

	// Fatal error or attempt limit reached: park in dead-letter state
	if isFatal || currentAttempt >= event.MaxAttempts {
		event.MarkDeadLetter(executionErr, now)

		slog.ErrorContext(finalizeCtx, "outbox event execution exhausted or fatal error; moving to dead letter",
			"event_id", event.ID,
			"event_type", event.Type,
			"attempt", currentAttempt,
			"max_attempts", event.MaxAttempts,
			"is_fatal", isFatal,
			"error", executionErr,
		)

		if dbErr := w.repo.MarkDeadLetter(finalizeCtx, event, w.id); dbErr != nil {
			slog.ErrorContext(finalizeCtx, "failed to dead letter outbox event",
				"event_id", event.ID,
				"worker_id", w.id,
				"error", dbErr,
			)
		}
		return
	}

	// Standard retry: domain method handles attempt increment and backoff calculation
	event.MarkFailure(executionErr, now)

	slog.WarnContext(finalizeCtx, "outbox event execution failed; scheduling retry",
		"event_id", event.ID,
		"event_type", event.Type,
		"attempt", event.Attempts, // Reflects incremented attempt count
		"max_attempts", event.MaxAttempts,
		"next_attempt_at", event.NextAttemptAt,
		"error", executionErr,
	)

	if dbErr := w.repo.MarkFailure(finalizeCtx, event, w.id); dbErr != nil {
		slog.ErrorContext(finalizeCtx, "failed to record outbox failure state",
			"event_id", event.ID,
			"worker_id", w.id,
			"error", dbErr,
		)
	}
}

// startHeartbeat periodically extends the database lease while processing tasks.
func (w *Worker) startHeartbeat(
	parentCtx context.Context,
	event *Event,
	done <-chan struct{},
	cancelTask context.CancelCauseFunc,
) {
	// Calculate half of the lease duration for the heartbeat interval
	interval := w.leaseDuration / 2
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
			renewCtx, cancel := context.WithTimeout(context.WithoutCancel(parentCtx), 2*time.Second)

			now := time.Now()
			newLease := now.Add(w.leaseDuration)

			if err := w.repo.RenewLease(renewCtx, event.ID, w.id, newLease, now); err != nil {
				slog.WarnContext(renewCtx, "failed to renew outbox event lease; cancelling task context",
					"event_id", event.ID,
					"worker_id", w.id,
					"error", err,
				)

				cancelTask(fmt.Errorf("lease renewal failed: %w", err))
				cancel()
				return
			}
			cancel()
		}
	}
}
