package httpio

import (
	"bonfire-api/internal/errs"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const (
	contentTypeJSON = "application/json"
)

var bufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, 2048))
	},
}

type Error struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Status  string        `json:"status"`
	Details []errs.Detail `json:"details,omitempty"`
}

type ErrorResponse struct {
	Error Error `json:"error"`
}

func RespondOK[T any](w http.ResponseWriter, r *http.Request, data T) {
	respondJSON(w, r, http.StatusOK, data)
}

func RespondCreated[T any](w http.ResponseWriter, r *http.Request, data T) {
	respondJSON(w, r, http.StatusCreated, data)
}

func RespondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func respondJSON(w http.ResponseWriter, r *http.Request, status int, data any) {
	writeJSON(w, r, status, contentTypeJSON, data)
}

func ToHTTPErr(h func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			respondError(w, r, err)
		}
	}
}

func respondError(w http.ResponseWriter, r *http.Request, err error) {
	var e *errs.Error

	switch {
	case errors.As(err, &e):
	case errors.Is(err, context.DeadlineExceeded):
		e = errs.DeadlineExceeded("Request timed out processing.").Wrap(err)
	case errors.Is(err, context.Canceled):
		e = errs.Aborted("Client closed connection mid-request.").Wrap(err)
	default:
		e = errs.Internal("An unexpected error occurred.").Wrap(err)
	}

	httpCode := e.Code.HTTPStatus()

	publicMsg := e.Message
	if httpCode >= 500 {
		publicMsg = errs.CodeInternal.Message()
	}

	respPayload := ErrorResponse{
		Error: Error{
			Code:    httpCode,
			Message: publicMsg,
			Status:  e.Code.String(),
			Details: e.Details,
		},
	}

	logError(r, e, httpCode)
	writeJSON(w, r, httpCode, contentTypeJSON, respPayload)
}

func writeJSON(w http.ResponseWriter, r *http.Request, status int, contentType string, data any) {
	const maxPoolBufferCapacity = 64 * 1024
	ctx := r.Context()

	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()

	defer func() {
		if buf.Cap() <= maxPoolBufferCapacity {
			bufferPool.Put(buf)
		}
	}()

	if err := json.NewEncoder(buf).Encode(data); err != nil {
		slog.ErrorContext(ctx, "failed to encode json response payload",
			"error", err,
			"http.path", r.URL.Path,
		)

		w.Header().Set("Content-Type", contentTypeJSON)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"code":500,"message":"An unexpected error occurred during payload encoding.","status":"INTERNAL"}}`))
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func logError(r *http.Request, e *errs.Error, httpCode int) {
	level := slog.LevelInfo
	if httpCode >= 500 {
		level = slog.LevelError
	} else if httpCode >= 400 {
		level = slog.LevelWarn
	}

	attrs := []any{
		"http.method", r.Method,
		"http.path", r.URL.Path,
		"http.status_code", httpCode,
		"error.code", e.Code.String(),
		"error.message", e.Message,
		"time", time.Now().UTC().Format(time.RFC3339),
	}

	if unwrapped := e.Unwrap(); unwrapped != nil {
		attrs = append(attrs, "error.raw", unwrapped.Error())
	}

	if len(e.Details) > 0 {
		attrs = append(attrs, "error.details", e.Details)
	}

	slog.Log(r.Context(), level, "http request execution failed", attrs...)
}
