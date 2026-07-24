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

var ErrFatal = errors.New("fatal outbox error")

type Handler func(ctx context.Context, payload json.RawMessage) error

type Repository interface {
	AcquireBatch(ctx context.Context, workerID uuid.UUID, leaseDurationSec int32, batchSize int32) ([]*Event, error)
	Create(ctx context.Context, event *Event) error
	Delete(ctx context.Context, id EventID) error
	Get(ctx context.Context, id EventID) (*Event, error)
	List(ctx context.Context, cursorID *EventID, limit int32) ([]*Event, error)
	MarkDeadLetter(ctx context.Context, id EventID, reason string) (*Event, error)
	MarkProcessed(ctx context.Context, id EventID) (*Event, error)
	Publish(ctx context.Context, variant string, payload any) (*Event, error)
	PurgeProcessed(ctx context.Context, retentionDays int32) error
	RecordFailure(ctx context.Context, id EventID, lastError string) (*Event, error)
	RenewLease(ctx context.Context, id EventID, workerID uuid.UUID, leaseDurationSec int32) error
	Save(ctx context.Context, event *Event) error
}

type Worker struct {
	id            uuid.UUID
	repository    Repository
	pollInterval  time.Duration
	leaseDuration int32
	batchSize     int32
	handlers      map[string]Handler
	handlersMu    sync.RWMutex
	wg            sync.WaitGroup
	cancel        context.CancelFunc
}

func NewWorker(
	repository Repository,
	pollInterval time.Duration,
	leaseDuration int32,
	batchSize int32,
) *Worker {
	return &Worker{
		id:            uuid.New(),
		repository:    repository,
		pollInterval:  pollInterval,
		leaseDuration: leaseDuration,
		batchSize:     batchSize,
		handlers:      make(map[string]Handler),
	}
}

// RegisterHandler registers a callback function for a specific event type string.
func (w *Worker) RegisterHandler(eventType string, handler Handler) {
	w.handlersMu.Lock()
	defer w.handlersMu.Unlock()
	w.handlers[eventType] = handler
}

func (w *Worker) Start(ctx context.Context) {
	workerCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(workerCtx, "recovered from panic in outbox worker goroutine", "panic", r)
			}
		}()

		slog.InfoContext(workerCtx, "initializing background outbox processor", "worker_id", w.id)

		ticker := time.NewTicker(w.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				w.processBatch(workerCtx)
			case <-workerCtx.Done():
				slog.InfoContext(workerCtx, "system cancellation detected; stopping outbox worker loop")
				return
			}
		}
	}()
}

func (w *Worker) Stop() {
	if w.cancel != nil {
		w.cancel()
		w.wg.Wait()
		slog.Info("outbox background processor gracefully stopped")
	}
}

func (w *Worker) processBatch(ctx context.Context) {
	events, err := w.repository.AcquireBatch(ctx, AcquireBatchParams{
		Limit:                  w.batchSize,
		WorkerID:               w.id,
		LeaseDurationInSeconds: w.leaseDuration,
	})
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.ErrorContext(ctx, "failed to acquire outbox events", "error", err)
		}
		return
	}

	for _, event := range events {
		if ctx.Err() != nil {
			return
		}
		w.executeEvent(ctx, event)
	}
}

func (w *Worker) executeEvent(ctx context.Context, event Event) {
	w.handlersMu.RLock()
	handler, exists := w.handlers[event.EventType]
	w.handlersMu.RUnlock()

	if !exists {
		slog.WarnContext(ctx, "unhandled event type dropped", "event_type", event.EventType, "event_id", event.ID)
		w.handleFailure(ctx, event, fmt.Errorf("no handler registered for event type: %s", event.EventType), true)
		return
	}

	executionErr := handler(ctx, event.Payload)
	isFatal := false

	if executionErr != nil {
		if errors.Is(executionErr, context.Canceled) {
			slog.InfoContext(ctx, "execution aborted due to shutdown; relying on lease expiration", "event_id", event.ID)
			return
		}
		w.handleFailure(ctx, event, executionErr, isFatal)
		return
	}

	finalizeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err := w.repository.MarkProcessed(finalizeCtx, event.ID); err != nil {
		slog.ErrorContext(finalizeCtx, "failed to finalize successful outbox event", "event_id", event.ID, "error", err)
		return
	}

	slog.InfoContext(finalizeCtx, "successfully processed outbox event", "event_id", event.ID, "event_type", event.EventType)
}

func (w *Worker) handleFailure(ctx context.Context, event Event, err error, isFatal bool) {
	finalizeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	logCtx := ctx
	if logCtx.Err() != nil {
		logCtx = finalizeCtx
	}

	if isFatal || (event.Attempts+1) >= event.MaxAttempts {
		slog.ErrorContext(logCtx, "outbox event processing exhausted; routing to dead letter",
			"event_id", event.ID,
			"event_type", event.EventType,
			"error", err,
		)

		_, dbErr := w.repository.MarkDeadLetter(finalizeCtx, MarkDeadLetterParams{
			ID:     event.ID,
			Reason: err.Error(),
		})
		if dbErr != nil {
			slog.ErrorContext(finalizeCtx, "failed to mark outbox event as dead letter", "event_id", event.ID, "error", dbErr)
		}
	} else {
		slog.WarnContext(logCtx, "outbox event retry registered",
			"event_id", event.ID,
			"attempt", event.Attempts+1,
			"error", err,
		)

		_, dbErr := w.repository.RecordFailure(finalizeCtx, RecordFailureParams{
			ID:        event.ID,
			LastError: err.Error(),
		})
		if dbErr != nil {
			slog.ErrorContext(finalizeCtx, "failed to record outbox failure state to database", "event_id", event.ID, "error", dbErr)
		}
	}
}

// func main() {
//     // ...
//     worker := NewWorker(outboxRepo, 5*time.Second, 60, 10)

//     // Register Auth Handlers
//     worker.RegisterHandler(auth.EventForgotPassword, func(ctx context.Context, raw json.RawMessage) error {
//         var p auth.ForgotPasswordPayload
//         if err := json.Unmarshal(raw, &p); err != nil {
//             return fmt.Errorf("malformed forgot password payload: %w", err)
//         }
//         return mailer.SendPasswordResetEmail(ctx, p.Email, p.Token)
//     })

//     worker.RegisterHandler(auth.EventRegister, func(ctx context.Context, raw json.RawMessage) error {
//         var p auth.RegisterPayload
//         if err := json.Unmarshal(raw, &p); err != nil {
//             return fmt.Errorf("malformed register payload: %w", err)
//         }
//         return mailer.SendRegisterEmail(ctx, p.Email, p.Username, p.Token)
//     })

//     worker.Start(ctx)
// }
