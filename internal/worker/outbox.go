package worker

import (
	"bonfire-api/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Mailer abstracts our email delivery engine (e.g., SendGrid, MockMailer, AWS SES)
type Mailer interface {
	SendWelcomeEmail(ctx context.Context, email string, username string, token string) error
	SendPasswordResetEmail(ctx context.Context, email string, resetToken string) error
}

type OutboxWorker struct {
	store     *repository.Queries
	mailer    Mailer
	ticker    *time.Ticker
	stopChan  chan struct{}
	batchSize int32
}

func NewOutboxWorker(store *repository.Queries, mailer Mailer, pollInterval time.Duration, batchSize int32) *OutboxWorker {
	return &OutboxWorker{
		store:     store,
		mailer:    mailer,
		ticker:    time.NewTicker(pollInterval),
		stopChan:  make(chan struct{}),
		batchSize: batchSize,
	}
}

// Start now accepts the global context to orchestrate lifecycle shutdown.
func (w *OutboxWorker) Start(ctx context.Context) {
	log.Println("[WORKER] Initializing background outbox processor...")
	go func() {
		for {
			select {
			case <-w.ticker.C:
				w.processBatch(ctx)
			case <-ctx.Done():
				w.ticker.Stop()
				log.Println("[WORKER] System cancellation detected. Stopping outbox worker loop...")
				return
			case <-w.stopChan:
				w.ticker.Stop()
				log.Println("[WORKER] Explicit stop signaled. Stopping outbox worker loop...")
				return
			}
		}
	}()
}

// Stop safely cuts off the ticker loop during graceful container shutdowns.
func (w *OutboxWorker) Stop() {
	close(w.stopChan)
	log.Println("[WORKER] Outbox background processor gracefully stopped.")
}

func (w *OutboxWorker) processBatch(ctx context.Context) {
	// 1. Fetch an isolated, concurrency-locked slice of pending work using the live context
	events, err := w.store.OutboxEventListUnprocessed(ctx, w.batchSize)
	if err != nil {
		// Avoid logging errors if the query failed purely because the system is shutting down
		if !errors.Is(err, context.Canceled) {
			log.Printf("[WORKER ERROR] Failed to fetch outbox events: %v", err)
		}
		return
	}

	for _, event := range events {
		// Fail-fast check: If the application context cancelled mid-batch loop,
		// don't bother wasting execution cycles starting the next event.
		if ctx.Err() != nil {
			return
		}
		w.executeEvent(ctx, event)
	}
}

func (w *OutboxWorker) executeEvent(ctx context.Context, event repository.OutboxEventListUnprocessedRow) {
	var executionErr error

	// 2. Evaluate the event type signature
	switch event.EventType {
	case "user.registered":
		var payload struct {
			Email    string `json:"email"`
			Username string `json:"username"`
			Token    string `json:"token"`
		}

		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			w.handleFailure(ctx, event, err, true)
			return
		}

		executionErr = w.mailer.SendWelcomeEmail(ctx, payload.Email, payload.Username, payload.Token)

	case "user.forgot_password":
		var payload struct {
			Email string `json:"email"`
			Token string `json:"token"`
		}

		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			w.handleFailure(ctx, event, err, true)
			return
		}

		executionErr = w.mailer.SendPasswordResetEmail(ctx, payload.Email, payload.Token)

	default:
		log.Printf("[WORKER WARN] Unhandled event type dropped: %s", event.EventType)
		return
	}

	if executionErr != nil {
		w.handleFailure(ctx, event, executionErr, false)
		return
	}

	if err := w.store.OutboxEventMarkProcessed(ctx, event.ID); err != nil {
		log.Printf("[WORKER ERROR] Failed to finalize successful event %s: %v", event.ID, err)
	}
}

func (w *OutboxWorker) handleFailure(ctx context.Context, event repository.OutboxEventListUnprocessedRow, err error, isFatal bool) {
	const maxAttempts = 5

	// Only handle the "Fatal" or "Max Exceeded" case in Go
	if isFatal || (event.Attempts+1) >= maxAttempts {
		// Option A: Use a special "dead letter" SQL query
		// Option B: Set next_attempt_at to a distant future year (e.g., 9999)
		log.Printf("[WORKER DEAD LETTER] Event %s exhausted.", event.ID)
		// ... call dedicated DeadLetterUpdate query ...
	} else {
		// Let the SQL query handle the exponential math automatically
		err = w.store.OutboxEventRecordFailure(ctx, repository.OutboxEventRecordFailureParams{
			ID:        event.ID,
			LastError: pgtype.Text{String: err.Error(), Valid: true},
		})
	}
}
