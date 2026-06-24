package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"bonfire-api/internal/repository"

	"github.com/jackc/pgx/v5/pgtype"
)

// --- OUTBOX WORKER TYPES ---

// OutboxWorker
type OutboxWorker struct {
	store        *repository.Queries
	pollInterval time.Duration
	batchSize    int32
	wg           sync.WaitGroup
	cancel       context.CancelFunc
}

// --- OUTBOX WORKER INITIALIZATION ---

// NewOutboxWorker
func NewOutboxWorker(store *repository.Queries, pollInterval time.Duration, batchSize int32) *OutboxWorker {
	return &OutboxWorker{
		store:        store,
		pollInterval: pollInterval,
		batchSize:    batchSize,
	}
}

// --- OUTBOX WORKER METHODS ---

// Start spawns the background event processing loop.
func (w *OutboxWorker) Start(ctx context.Context) {
	// Create a dedicated context for the worker loop
	workerCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		// Panic recovery for background worker robustness
		defer func() {
			if r := recover(); r != nil {
				slog.Error("recovered from panic in outbox worker goroutine", "panic", r)
			}
		}()

		slog.Info("initializing background outbox processor")

		// Use a dynamic timer rather than a ticker to prevent overlap
		timer := time.NewTimer(w.pollInterval)
		defer timer.Stop()

		for {
			select {
			case <-timer.C:
				w.processBatch(workerCtx)
				// Reset the timer only after the batch has finished processing
				timer.Reset(w.pollInterval)
			case <-workerCtx.Done():
				slog.Info("system cancellation detected; stopping outbox worker loop")
				return
			}
		}
	}()
}

// Stop gracefully shuts down the worker, waiting for the current batch to finish.
func (w *OutboxWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
		w.wg.Wait() // Block until the active batch completes
		slog.Info("outbox background processor gracefully stopped")
	}
}

// processBatch fetches and processes a single window of events.
func (w *OutboxWorker) processBatch(ctx context.Context) {
	// Using AcquireBatch safely leases the items by updating their next_attempt_at
	events, err := w.store.OutboxEventAcquireBatch(ctx, w.batchSize)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.Error("failed to acquire outbox events", "error", err)
		}
		return
	}

	for _, event := range events {
		// Stop processing remaining items if shutting down
		if ctx.Err() != nil {
			return
		}
		w.executeEvent(ctx, event)
	}
}

// executeEvent routes an individual event payload depending on its type signature.
func (w *OutboxWorker) executeEvent(ctx context.Context, event repository.OutboxEvent) {
	var executionErr error
	var isFatal bool

	switch event.EventType {
	case "user.registered":
		var payload RegisterEventPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			executionErr, isFatal = err, true
			break
		}

		if !payload.UserID.Valid {
			executionErr, isFatal = errors.New("invalid or missing user_id in payload"), true
			break
		}

		_, executionErr = w.store.UserMarkVerified(ctx, payload.UserID)

	// Add other cases here...

	default:
		slog.Warn("unhandled event type dropped", "event_type", event.EventType, "event_id", event.ID)
		// Mark as dead letter to avoid reprocessing unhandled events
		executionErr, isFatal = errors.New("unhandled event type"), true
	}

	// Handle routing based on success/failure
	if executionErr != nil {
		if errors.Is(executionErr, context.Canceled) {
			slog.Info("execution aborted due to shutdown; relying on lease expiration", "event_id", event.ID)
			return
		}

		w.handleFailure(event, executionErr, isFatal)
		return
	}

	// Use a background context with a short timeout to ensure the success
	// state is saved even if the worker context was canceled right at this exact line.
	finalizeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err := w.store.OutboxEventMarkProcessed(finalizeCtx, event.ID); err != nil {
		slog.Error("failed to finalize successful outbox event", "event_id", event.ID, "error", err)
	}

	slog.Info("successfully processed outbox event",
		"event_id", event.ID,
		"event_type", event.EventType,
	)
}

// handleFailure logs processing errors and updates the database state accordingly.
func (w *OutboxWorker) handleFailure(event repository.OutboxEvent, err error, isFatal bool) {
	finalizeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Rely on the database struct for max attempts rather than a hardcoded constant
	if isFatal || (event.Attempts+1) >= event.MaxAttempts {
		slog.Error("outbox event processing exhausted; routing to dead letter",
			"event_id", event.ID,
			"event_type", event.EventType,
			"error", err,
		)

		// Actually update the database so it stops fetching this event
		_, dbErr := w.store.OutboxEventMarkDeadLetter(finalizeCtx, repository.OutboxEventMarkDeadLetterParams{
			ID:        event.ID,
			LastError: pgtype.Text{String: err.Error(), Valid: true},
		})
		if dbErr != nil {
			slog.Error("failed to mark outbox event as dead letter", "event_id", event.ID, "error", dbErr)
		}
	} else {
		slog.Warn("outbox event retry registered",
			"event_id", event.ID,
			"attempt", event.Attempts+1,
			"error", err,
		)

		_, dbErr := w.store.OutboxEventRecordFailure(finalizeCtx, repository.OutboxEventRecordFailureParams{
			ID:        event.ID,
			LastError: pgtype.Text{String: err.Error(), Valid: true},
		})
		if dbErr != nil {
			slog.Error("failed to record outbox failure state to database", "event_id", event.ID, "error", dbErr)
		}
	}
}
