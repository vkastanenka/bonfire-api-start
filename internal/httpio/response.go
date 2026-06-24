package httpio

import (
	"bonfire-api/internal/apperr"
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5/middleware"
)

// MaxPoolBufferCapacity prevents oversized buffers from polluting memory (64 KB)
const maxPoolBufferCapacity = 64 * 1024

var bufferPool = sync.Pool{
	New: func() interface{} {
		// Pre-allocate a reasonable starting size (e.g., 2 KB) to avoid early allocations
		return bytes.NewBuffer(make([]byte, 0, 2048))
	},
}

// RespondJSON marshals data and writes it securely to the wire.
// Pass r.Context() to ensure structured logs maintain distributed trace IDs.
func RespondJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	ctx := r.Context()

	w.Header().Set("Content-Type", "application/json")

	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()

	// Conditionally return the buffer to the pool to prevent memory bloat
	defer func() {
		if buf.Cap() <= maxPoolBufferCapacity {
			bufferPool.Put(buf)
		}
	}()

	if err := json.NewEncoder(buf).Encode(data); err != nil {
		// Uses ErrorContext to tie logs directly to your request span/ID
		slog.ErrorContext(ctx, "failed to encode json response", "error", err)

		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"INTERNAL","message":"An unexpected error occurred."}`))
		return
	}

	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

// ==========================================
// STANDARD ENVELOPES
// ==========================================

// SuccessResponse defines the standard envelope for all successful API responses.
type SuccessResponse[T any] struct {
	Message string `json:"message,omitempty"` // Optional human-readable message
	Data    T      `json:"data"`              // The actual payload
	Meta    any    `json:"meta,omitempty"`    // Pagination, cursors, or extra context
}

// OffsetPagination defines the meta payload for page-based lists.
type OffsetPagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// CursorPagination defines the metadata returned for keyset-paginated lists.
type CursorPagination struct {
	NextCursor *string `json:"next_cursor,omitempty"` // Opaque string for the client
	PageSize   int32   `json:"page_size"`             // Count of items in this specific batch
}

func RespondOK[T any](w http.ResponseWriter, r *http.Request, data T, message string) {
	RespondJSON(w, r, http.StatusOK, SuccessResponse[T]{
		Message: message,
		Data:    data,
	})
}

func RespondCreated[T any](w http.ResponseWriter, r *http.Request, data T, message string) {
	RespondJSON(w, r, http.StatusCreated, SuccessResponse[T]{
		Message: message,
		Data:    data,
	})
}

func RespondCursorList[T any](w http.ResponseWriter, r *http.Request, data T, message string, meta CursorPagination) {
	RespondJSON(w, r, http.StatusOK, SuccessResponse[T]{
		Data: data,
		Meta: meta,
	})
}

func RespondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// TODO: Deprecate
func RespondText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// TODO: Deprecate
func RespondTextError(w http.ResponseWriter, r *http.Request, logMsg string, err error, status int, userMsg string) {
	reqID := middleware.GetReqID(r.Context())
	slog.ErrorContext(r.Context(), logMsg, "error", err, "reqID", reqID)
	RespondText(w, status, userMsg)
}

// ToHTTP wraps handlers that return an error to centralize response/logging logic
func ToHTTP(h func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err == nil {
			return
		}

		// Identify/Normalize error
		var appErr *apperr.Error
		if !errors.As(err, &appErr) {
			err = apperr.New(apperr.CodeInternal, apperr.CodeInternal.Title(), apperr.WithErr(err))
		}

		// Enrich and Prepare
		appErr.Enrich(r)
		status, resp := appErr.HTTPResponse()

		// Log
		logError(r, appErr, err, status)

		// Respond
		RespondJSON(w, r, status, resp)
	}
}

// logError logs app errors
func logError(r *http.Request, appErr *apperr.Error, originalErr error, status int) {
	// level := slog.LevelInfo
	// if appErr.IsCode(apperr.CodeInternal) {
	// 	level = slog.LevelError
	// }

	// args := []any{
	// 	"path", r.URL.Path,
	// 	"method", r.Method,
	// 	"status", status,
	// 	slog.Group("error_context",
	// 		"code", appErr.Code,
	// 		"request_id", appErr.RequestID,
	// 		"trace_id", appErr.TraceID,
	// 		"error", originalErr,
	// 	),
	// }

	// if len(appErr.Details) > 0 {
	// 	args = append(args, "details", appErr.Details)
	// }
	// if len(appErr.ValidationErrors) > 0 {
	// 	args = append(args, "validation_errors", appErr.ValidationErrors)
	// }

	// slog.Log(r.Context(), level, HTTPReqFailedMsg, args...)
}
