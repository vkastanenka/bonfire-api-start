package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"bonfire-api/internal/cache"
	"bonfire-api/internal/email"

	"github.com/google/uuid"
)

type Worker struct {
	id            uuid.UUID
	cache         cache.Manager
	service       *Service
	mailer        email.Mailer
	pollInterval  time.Duration
	leaseDuration int32
	batchSize     int32
	wg            sync.WaitGroup
	cancel        context.CancelFunc
}

func NewWorker(
	cache cache.Manager,
	service *Service,
	mailer email.Mailer,
	pollInterval time.Duration,
	leaseDuration int32,
	batchSize int32,
) *Worker {
	return &Worker{
		id:            uuid.New(),
		cache:         cache,
		service:       service,
		mailer:        mailer,
		pollInterval:  pollInterval,
		leaseDuration: leaseDuration,
		batchSize:     batchSize,
	}
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
	events, err := w.service.AcquireBatch(ctx, AcquireBatchParams{
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
	var executionErr error
	var isFatal bool

	switch Type(event.EventType) {
	case EventAuthRegister:
		var payload RegisterPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			executionErr, isFatal = err, true
			break
		}
		executionErr = w.mailer.SendRegisterEmail(ctx, payload.Email, payload.Username, payload.Token)

	case EventAuthResendVerification:
		var payload ResendVerificationPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			executionErr, isFatal = err, true
			break
		}
		executionErr = w.mailer.SendRegisterEmail(ctx, payload.Email, payload.Username, payload.Token)

	case EventAuthForgotPassword:
		var payload ForgotPasswordPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			executionErr, isFatal = err, true
			break
		}
		executionErr = w.mailer.SendPasswordResetEmail(ctx, payload.Email, payload.Token)

	case EventPresenceUpdated:
		var payload PresenceUpdatedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			executionErr, isFatal = err, true
			break
		}

		userIdStr := uuid.MustParse(payload.UserID)
		presenceEnum := cache.ParsePresence(payload.Presence)

		if err := w.cache.Heartbeat(ctx, userIdStr, presenceEnum); err != nil {
			executionErr = fmt.Errorf("failed to sync heartbeat to redis: %w", err)
			break
		}

		if err := w.cache.Publish(ctx, "presence:updated", payload); err != nil {
			executionErr = fmt.Errorf("failed to broadcast presence pubsub: %w", err)
		}

	default:
		slog.WarnContext(ctx, "unhandled event type dropped", "event_type", event.EventType, "event_id", event.ID)
		executionErr, isFatal = errors.New("unhandled event type"), true
	}

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

	if _, err := w.service.MarkProcessed(finalizeCtx, event.ID); err != nil {
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

		_, dbErr := w.service.MarkDeadLetter(finalizeCtx, MarkDeadLetterParams{
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

		_, dbErr := w.service.RecordFailure(finalizeCtx, RecordFailureParams{
			ID:        event.ID,
			LastError: err.Error(),
		})
		if dbErr != nil {
			slog.ErrorContext(finalizeCtx, "failed to record outbox failure state to database", "event_id", event.ID, "error", dbErr)
		}
	}
}
