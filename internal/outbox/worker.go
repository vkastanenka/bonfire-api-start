package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	defaultPollInterval      = 2 * time.Second
	defaultLeaseDuration     = 30 * time.Second
	defaultBatchSize         = 50
	defaultMaxWorkers        = 10
	defaultMaxAttempts       = 3
	defaultHandlerTimeout    = 5 * time.Second
	defaultHeartbeatInterval = 2 * time.Second
	maxLastErrorLen          = 4096
)

type Worker struct {
	id            uuid.UUID
	repo          *Repository
	pollInterval  time.Duration
	leaseDuration time.Duration
	batchSize     int
	maxWorkers    int
	sem           chan struct{}
	handlers      map[string]Handler
	wg            sync.WaitGroup
	cancel        context.CancelFunc
}

func NewWorker(
	repo *Repository,
	pollInterval time.Duration,
	leaseDuration time.Duration,
	batchSize int,
	maxWorkers int,
) (*Worker, error) {
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}
	if leaseDuration <= 0 {
		leaseDuration = defaultLeaseDuration
	}
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	if maxWorkers <= 0 {
		maxWorkers = defaultMaxWorkers
	}

	return &Worker{
		id:            uuid.New(),
		repo:          repo,
		pollInterval:  pollInterval,
		leaseDuration: leaseDuration,
		batchSize:     batchSize,
		maxWorkers:    maxWorkers,
		sem:           make(chan struct{}, maxWorkers),
		handlers:      make(map[string]Handler),
	}, nil
}

// RegisterHandler registers a callback function for a specific event type.
func (w *Worker) RegisterHandler(eventType string, handler Handler) {
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

		slog.InfoContext(workerCtx,
			"initializing outbox worker",
			"worker_id", w.id,
		)

		w.processBatch(workerCtx)

		ticker := time.NewTicker(w.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				w.processBatch(workerCtx)
			case <-workerCtx.Done():
				slog.InfoContext(workerCtx,
					"stopping outbox worker",
					"worker_id", w.id,
				)
				return
			}
		}
	}()
}

// Stop signals the worker loop to cancel and waits for active routines to finish.
func (w *Worker) Stop(ctx context.Context) {
	if w.cancel == nil {
		return
	}

	w.cancel()

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.InfoContext(ctx,
			"outbox worker stopped",
			"worker_id", w.id,
		)
	case <-ctx.Done():
		slog.WarnContext(ctx,
			"outbox worker shutdown timed out or was canceled",
			"worker_id", w.id,
			"err", ctx.Err(),
		)
	}
}

// processBatch claims pending events and executes them concurrently using worker slots.
func (w *Worker) processBatch(ctx context.Context) {
	now := time.Now()

	events, err := w.repo.ClaimPending(ctx, w.id, now.Add(w.leaseDuration), now, w.batchSize)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.ErrorContext(ctx,
				"failed to claim outbox events",
				"error", err,
			)
		}
		return
	}

	if len(events) == 0 {
		return
	}

	var batchWg sync.WaitGroup

eventLoop:
	for _, evt := range events {
		select {
		case w.sem <- struct{}{}:
			batchWg.Add(1)
			go func(e *Event) {
				defer batchWg.Done()
				defer func() { <-w.sem }()
				w.executeEvent(ctx, e)
			}(evt)

		case <-ctx.Done():
			slog.InfoContext(ctx,
				"batch processing canceled; stopping further dispatches",
				"worker_id", w.id,
			)
			break eventLoop
		}
	}

	batchWg.Wait()
}

// executeEvent processes a single outbox event with lease management, context propagation, and error handling.
func (w *Worker) executeEvent(ctx context.Context, event *Event) {
	if event == nil {
		return
	}

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

	handler, exists := w.handlers[event.Type]
	if !exists {
		slog.WarnContext(ctx, "unknown outbox event type encountered in event execution",
			"event_id", event.ID,
			"event_type", event.Type,
		)
		w.handleFailure(ctx, event, fmt.Errorf("no handler registered for event type: %s", event.Type), true)
		return
	}

	timeoutCtx, cancelTimeout := context.WithTimeout(context.WithoutCancel(ctx), w.calculateHandlerTimeout())
	defer cancelTimeout()

	taskCtx, cancelTask := context.WithCancelCause(timeoutCtx)
	defer cancelTask(nil)

	heartbeatDone := make(chan struct{})
	go w.startHeartbeat(ctx, event, heartbeatDone, cancelTask)

	executionErr := handler(taskCtx, event.Payload)

	close(heartbeatDone)

	if executionErr != nil {
		if leaseErr := context.Cause(taskCtx); leaseErr != nil && errors.Is(leaseErr, ErrLeaseLost) {
			slog.ErrorContext(ctx, "handler execution aborted due to lost outbox lease",
				"event_id", event.ID,
				"error", leaseErr,
			)
			return
		}

		if errors.Is(executionErr, context.Canceled) && ctx.Err() != nil {
			slog.InfoContext(ctx, "execution context canceled during shutdown; releasing lease",
				"event_id", event.ID,
			)

			releaseCtx, cancelRelease := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
			defer cancelRelease()

			if err := w.repo.ReleaseLease(releaseCtx, event, w.id); err != nil {
				slog.ErrorContext(releaseCtx, "failed to release outbox event lease during shutdown",
					"event_id", event.ID,
					"worker_id", w.id,
					"error", err,
				)
			}
			return
		}

		isFatal := errors.Is(executionErr, ErrFatal) || errors.Is(executionErr, ErrInvalidPayload)
		w.handleFailure(ctx, event, executionErr, isFatal)
		return
	}

	event.MarkProcessed(time.Now())

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

// handleFailure transitions an event to either dead-letter or schedules backoff retries.
func (w *Worker) handleFailure(ctx context.Context, event *Event, executionErr error, isFatal bool) {
	now := time.Now()

	finalizeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()

	currentAttempt := event.Attempts + 1

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

	event.MarkFailure(executionErr, now)

	slog.WarnContext(finalizeCtx, "outbox event execution failed; scheduling retry",
		"event_id", event.ID,
		"event_type", event.Type,
		"attempt", event.Attempts,
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

// startHeartbeat extends the outbox database lease while processing tasks.
func (w *Worker) startHeartbeat(
	parentCtx context.Context,
	event *Event,
	done <-chan struct{},
	cancelTask context.CancelCauseFunc,
) {
	ticker := time.NewTicker(w.calculateHeartbeatInterval())
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-parentCtx.Done():
			return
		case <-ticker.C:
			select {
			case <-done:
				return
			default:
			}

			renewCtx, cancel := context.WithTimeout(context.WithoutCancel(parentCtx), 2*time.Second)
			now := time.Now()
			newLease := now.Add(w.leaseDuration)

			err := w.repo.RenewLease(renewCtx, event.ID, w.id, newLease, now)
			cancel()

			if err != nil {
				slog.WarnContext(parentCtx, "failed to renew outbox event lease; cancelling task context",
					"event_id", event.ID,
					"worker_id", w.id,
					"error", err,
				)
				cancelTask(fmt.Errorf("%w: %w", ErrLeaseLost, err))
				return
			}
		}
	}
}

// calculateHandlerTimeout computes a timeout based on the lease duration.
func (w *Worker) calculateHandlerTimeout() time.Duration {
	timeout := time.Duration(float64(w.leaseDuration) * 0.8)
	if timeout <= 0 {
		return defaultHandlerTimeout
	}
	return timeout
}

// calculateHeartbeatInterval computes a safe heartbeat interval based on the lease duration.
func (w *Worker) calculateHeartbeatInterval() time.Duration {
	interval := w.leaseDuration / 2
	if interval < defaultHeartbeatInterval {
		return defaultHeartbeatInterval
	}
	return interval
}
