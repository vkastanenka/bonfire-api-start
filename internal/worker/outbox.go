package worker

import (
	"bonfire-api/internal/repository"
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Mailer abstracts our email delivery engine (e.g., SendGrid, MockMailer, AWS SES)
type Mailer interface {
	SendWelcomeEmail(ctx context.Context, email string, username string) error
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

// Start boots the asynchronous polling loop in its own persistent background goroutine.
func (w *OutboxWorker) Start() {
	log.Println("[WORKER] Initializing background outbox processor...")
	go func() {
		for {
			select {
			case <-w.ticker.C:
				w.processBatch()
			case <-w.stopChan:
				w.ticker.Stop()
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

func (w *OutboxWorker) processBatch() {
	// Use a clean, un-canceled root context for persistent background processing
	ctx := context.Background()

	// 1. Fetch an isolated, concurrency-locked slice of pending work
	events, err := w.store.GetUnprocessedOutboxEvents(ctx, w.batchSize)
	if err != nil {
		log.Printf("[WORKER ERROR] Failed to fetch outbox events: %v", err)
		return
	}

	for _, event := range events {
		w.executeEvent(ctx, event)
	}
}

func (w *OutboxWorker) executeEvent(ctx context.Context, event repository.GetUnprocessedOutboxEventsRow) {
	var executionErr error

	// 2. Evaluate the event type signature
	switch event.EventType {
	case "user.registered":
		var payload struct {
			Email    string `json:"email"`
			Username string `json:"username"`
		}

		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			w.handleFailure(ctx, event, err, true)
			return
		}

		executionErr = w.mailer.SendWelcomeEmail(ctx, payload.Email, payload.Username)

	default:
		log.Printf("[WORKER WARN] Unhandled event type dropped: %s", event.EventType)
		return
	}

	if executionErr != nil {
		w.handleFailure(ctx, event, executionErr, false)
		return
	}

	if err := w.store.MarkOutboxEventProcessed(ctx, event.ID); err != nil {
		log.Printf("[WORKER ERROR] Failed to finalize successful event %s: %v", event.ID, err)
	}
}

func (w *OutboxWorker) handleFailure(ctx context.Context, event repository.GetUnprocessedOutboxEventsRow, err error, isFatal bool) {
	log.Printf("[WORKER EXECUTION FAILURE] Event ID: %s, Error: %v", event.ID, err)

	var nextAttempt time.Time
	currentAttempts := event.Attempts + 1
	const maxAttempts = 5 // Matches your database DDL default constraint ceiling

	if isFatal || currentAttempts >= maxAttempts {
		nextAttempt = time.Now().Add(100 * 365 * 24 * time.Hour)
		log.Printf("[WORKER DEAD LETTER] Event %s completely exhausted retries. Moved to dead letter state.", event.ID)
	} else {
		backoffDuration := time.Duration(1<<uint(currentAttempts)) * time.Minute
		nextAttempt = time.Now().Add(backoffDuration)
		log.Printf("[WORKER RETRY SCHEDULED] Event %s scheduled for retry in %v", event.ID, backoffDuration)
	}

	_ = w.store.RecordOutboxEventFailure(ctx, repository.RecordOutboxEventFailureParams{
		ID:        event.ID,
		LastError: pgtype.Text{String: err.Error(), Valid: true},
		NextAttemptAt: pgtype.Timestamptz{
			Time:  nextAttempt,
			Valid: true,
		},
	})
}
