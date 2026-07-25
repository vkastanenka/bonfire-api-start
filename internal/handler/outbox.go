package handler

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/outbox"
	"context"
	"net/http"

	"github.com/google/uuid"
)

type OutboxRepository interface {
	AcquireBatch(ctx context.Context, workerID uuid.UUID, leaseDurationSec int32, batchSize int32) ([]*outbox.Event, error)
	Create(ctx context.Context, event *outbox.Event) error
	Delete(ctx context.Context, id outbox.EventID) error
	Get(ctx context.Context, id outbox.EventID) (*outbox.Event, error)
	List(ctx context.Context, cursorID *outbox.EventID, limit int32) ([]*outbox.Event, error)
	MarkDeadLetter(ctx context.Context, id outbox.EventID, reason string) (*outbox.Event, error)
	MarkProcessed(ctx context.Context, id outbox.EventID) (*outbox.Event, error)
	PurgeProcessed(ctx context.Context, retentionDays int32) error
	RecordFailure(ctx context.Context, id outbox.EventID, lastError string) (*outbox.Event, error)
	RenewLease(ctx context.Context, id outbox.EventID, workerID uuid.UUID, leaseDurationSec int32) error
	Save(ctx context.Context, event *outbox.Event) error
}

type Outbox struct {
	repo OutboxRepository
	bind *httpio.Bind
}

func NewOutbox(repo OutboxRepository, bind *httpio.Bind) *Outbox {
	return &Outbox{repo: repo, bind: bind}
}

type OutboxEventGetByIDPath struct {
	ID uuid.UUID `path:"id" validate:"required,uuid"`
}

func (h *Outbox) OutboxEventGetByID(w http.ResponseWriter, r *http.Request) error {
	var path OutboxEventGetByIDPath
	err := h.bind.Path(r, &path)
	if err != nil {
		return err
	}

	outboxID, err := outbox.NewEventID(path.ID)
	if err != nil {
		return errs.InvalidArgument("Invalid ID.").
			Wrap(err)
	}

	event, err := h.repo.Get(r.Context(), outboxID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, ToOutboxEventResponse(*event))
	return nil
}

type OutboxEventDeleteByIDPath struct {
	ID uuid.UUID `path:"id" validate:"required,uuid"`
}

func (h *Outbox) Delete(w http.ResponseWriter, r *http.Request) error {
	var path OutboxEventDeleteByIDPath
	err := h.bind.Path(r, &path)
	if err != nil {
		return err
	}

	outboxID, err := outbox.NewEventID(path.ID)
	if err != nil {
		return errs.InvalidArgument("Invalid ID.").
			Wrap(err)
	}

	if err = h.repo.Delete(r.Context(), outboxID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
